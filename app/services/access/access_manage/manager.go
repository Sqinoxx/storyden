// Package access_manage runs the request and approval flow for access passes.
// Payment happens offline: a member requests a pass, pays the student council
// quoting their reference code, and someone with MANAGE_ACCESS activates it.
package access_manage

import (
	"context"
	"log/slog"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/Southclaws/opt"

	"github.com/Southclaws/storyden/app/resources/access"
	"github.com/Southclaws/storyden/app/resources/access/access_repo"
	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/account/account_querier"
	"github.com/Southclaws/storyden/app/resources/account/notification"
	"github.com/Southclaws/storyden/app/resources/audit"
	"github.com/Southclaws/storyden/app/resources/audit/audit_writer"
	"github.com/Southclaws/storyden/app/resources/datagraph"
	"github.com/Southclaws/storyden/app/resources/rbac"
	"github.com/Southclaws/storyden/app/services/authentication/session"
	"github.com/Southclaws/storyden/app/services/notification/notify"
)

var (
	ErrNotRequestable   = fault.New("pass does not accept requests", ftag.With(ftag.InvalidArgument))
	ErrNoPendingRequest = fault.New("no pending request", ftag.With(ftag.InvalidArgument))
	ErrNotActive        = fault.New("grant is not active", ftag.With(ftag.InvalidArgument))
	ErrAccountNotFound  = fault.New("account not found", ftag.With(ftag.NotFound))
)

type RemoveAction string

const (
	RemoveActionReject RemoveAction = "reject"
	RemoveActionRevoke RemoveAction = "revoke"
)

type Manager struct {
	logger         *slog.Logger
	repo           *access_repo.Repository
	accountQuerier *account_querier.Querier
	auditWriter    *audit_writer.Writer
	notifier       *notify.Notifier
}

func New(
	logger *slog.Logger,
	repo *access_repo.Repository,
	accountQuerier *account_querier.Querier,
	auditWriter *audit_writer.Writer,
	notifier *notify.Notifier,
) *Manager {
	return &Manager{
		logger:         logger,
		repo:           repo,
		accountQuerier: accountQuerier,
		auditWriter:    auditWriter,
		notifier:       notifier,
	}
}

type PassWithMembership struct {
	*access.Pass
	Membership opt.Optional[access.Grant]
}

// ListPasses returns every pass with the requester's own grant attached.
func (m *Manager) ListPasses(ctx context.Context) ([]*PassWithMembership, error) {
	passes, err := m.repo.ListPasses(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	mine := map[access.PassID]*access.Grant{}
	if accountID, ok := session.GetOptAccountID(ctx).Get(); ok {
		grants, err := m.repo.ListAccountGrants(ctx, accountID)
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}
		for _, g := range grants {
			mine[g.PassID] = g
		}
	}

	out := make([]*PassWithMembership, len(passes))
	for i, p := range passes {
		out[i] = &PassWithMembership{
			Pass:       p,
			Membership: opt.NewPtr(mine[p.ID]),
		}
	}

	return out, nil
}

func (m *Manager) withMembership(ctx context.Context, passID access.PassID, accountID account.AccountID) (*PassWithMembership, error) {
	p, err := m.repo.GetPass(ctx, passID)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	g, err := m.repo.GetGrant(ctx, accountID, passID)
	if err != nil {
		if ftag.Get(err) == ftag.NotFound {
			return &PassWithMembership{Pass: p}, nil
		}
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return &PassWithMembership{Pass: p, Membership: opt.NewPtr(g)}, nil
}

func (m *Manager) Request(ctx context.Context, passID access.PassID) (*PassWithMembership, error) {
	accountID, err := session.GetAccountID(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	current, err := m.withMembership(ctx, passID, accountID)
	if err != nil {
		return nil, err
	}

	if g, ok := current.Membership.Get(); ok {
		if g.IsEffective(time.Now()) || g.Status == access.GrantStatusRequested {
			return current, nil
		}
	}

	if !current.Requestable {
		return nil, fault.Wrap(ErrNotRequestable, fctx.With(ctx),
			fmsg.WithDesc("not requestable", "This pass cannot be requested here, please contact the student council directly."))
	}

	if _, err := m.repo.SetGrant(ctx, accountID, passID, access.GrantStatusRequested, access_repo.GrantOptions{}); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if err := m.notifyManagers(ctx, accountID); err != nil {
		m.logger.Error("failed to notify access managers", slog.String("error", err.Error()))
	}

	return m.withMembership(ctx, passID, accountID)
}

func (m *Manager) Withdraw(ctx context.Context, passID access.PassID) (*PassWithMembership, error) {
	accountID, err := session.GetAccountID(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	g, err := m.repo.GetGrant(ctx, accountID, passID)
	if err != nil {
		if ftag.Get(err) == ftag.NotFound {
			return nil, fault.Wrap(ErrNoPendingRequest, fctx.With(ctx))
		}
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if g.Status != access.GrantStatusRequested {
		return nil, fault.Wrap(ErrNoPendingRequest, fctx.With(ctx))
	}

	if err := m.repo.DeleteGrant(ctx, g.ID); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return m.withMembership(ctx, passID, accountID)
}

type GrantParams struct {
	ExpiresAt opt.Optional[time.Time]
	Note      opt.Optional[string]
}

func (m *Manager) Grant(ctx context.Context, handle string, passID access.PassID, params GrantParams) (*access.Grant, error) {
	managerID, err := session.GetAccountID(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	memberID, err := m.lookup(ctx, handle)
	if err != nil {
		return nil, err
	}

	if exp, ok := params.ExpiresAt.Get(); ok && !exp.After(time.Now()) {
		return nil, fault.New("expiry in the past",
			fctx.With(ctx),
			ftag.With(ftag.InvalidArgument),
			fmsg.WithDesc("expiry", "The expiry date must be in the future."))
	}

	g, err := m.repo.SetGrant(ctx, memberID, passID, access.GrantStatusActive, access_repo.GrantOptions{
		ExpiresAt: params.ExpiresAt,
		DecidedBy: opt.New(managerID),
		Note:      params.Note,
	})
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	m.recordAudit(ctx, managerID, g)

	if err := m.notifier.Send(ctx, memberID, opt.New(managerID), notification.EventAccessGranted, nil); err != nil {
		m.logger.Error("failed to notify member of access grant", slog.String("error", err.Error()))
	}

	return m.repo.GetGrantDetailed(ctx, memberID, passID)
}

func (m *Manager) Remove(ctx context.Context, handle string, passID access.PassID, action RemoveAction) (*access.Grant, error) {
	managerID, err := session.GetAccountID(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	memberID, err := m.lookup(ctx, handle)
	if err != nil {
		return nil, err
	}

	current, err := m.repo.GetGrant(ctx, memberID, passID)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	var status access.GrantStatus
	switch action {
	case RemoveActionReject:
		if current.Status != access.GrantStatusRequested {
			return nil, fault.Wrap(ErrNoPendingRequest, fctx.With(ctx))
		}
		status = access.GrantStatusRejected

	case RemoveActionRevoke:
		if current.Status != access.GrantStatusActive {
			return nil, fault.Wrap(ErrNotActive, fctx.With(ctx))
		}
		status = access.GrantStatusRevoked

	default:
		return nil, fault.New("unknown action", fctx.With(ctx), ftag.With(ftag.InvalidArgument))
	}

	g, err := m.repo.SetGrant(ctx, memberID, passID, status, access_repo.GrantOptions{
		DecidedBy: opt.New(managerID),
	})
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	m.recordAudit(ctx, managerID, g)

	if status == access.GrantStatusRejected {
		if err := m.notifier.Send(ctx, memberID, opt.New(managerID), notification.EventAccessRejected, nil); err != nil {
			m.logger.Error("failed to notify member of access rejection", slog.String("error", err.Error()))
		}
	}

	return m.repo.GetGrantDetailed(ctx, memberID, passID)
}

type ListParams struct {
	Status opt.Optional[string]
	PassID opt.Optional[access.PassID]
	Handle opt.Optional[string]
}

func (m *Manager) ListGrants(ctx context.Context, params ListParams) ([]*access.Grant, error) {
	filter := access_repo.GrantFilter{
		Status: params.Status,
		PassID: params.PassID,
	}

	if handle, ok := params.Handle.Get(); ok {
		id, err := m.lookup(ctx, handle)
		if err != nil {
			return nil, err
		}
		filter.AccountID = opt.New(id)
	}

	return m.repo.ListGrants(ctx, filter, time.Now())
}

func (m *Manager) lookup(ctx context.Context, handle string) (account.AccountID, error) {
	acc, exists, err := m.accountQuerier.LookupByHandle(ctx, handle)
	if err != nil {
		return account.AccountID{}, fault.Wrap(err, fctx.With(ctx))
	}
	if !exists {
		return account.AccountID{}, fault.Wrap(ErrAccountNotFound, fctx.With(ctx))
	}
	return acc.ID, nil
}

func (m *Manager) notifyManagers(ctx context.Context, requester account.AccountID) error {
	managers, err := m.accountQuerier.ListByHeldPermission(ctx, rbac.PermissionAdministrator, rbac.PermissionManageAccess)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	for _, acc := range managers {
		if err := m.notifier.Send(ctx, acc.ID, opt.New(requester), notification.EventAccessRequested, nil); err != nil {
			return fault.Wrap(err, fctx.With(ctx))
		}
	}

	return nil
}

func (m *Manager) recordAudit(ctx context.Context, managerID account.AccountID, g *access.Grant) {
	pass, err := m.repo.GetPass(ctx, g.PassID)
	if err != nil {
		m.logger.Error("failed to load pass for audit", slog.String("error", err.Error()))
		return
	}

	meta := map[string]any{
		"account_id": g.AccountID.String(),
		"pass_id":    g.PassID.String(),
		"pass_name":  pass.Name,
		"status":     g.DisplayStatus(time.Now()),
	}
	g.ExpiresAt.Call(func(t time.Time) { meta["expires_at"] = t.Format(time.RFC3339) })
	if g.Note != "" {
		meta["note"] = g.Note
	}

	if _, err := m.auditWriter.Create(ctx, audit.EventTypeAccessGrantUpdated, opt.New(managerID), opt.NewEmpty[datagraph.Ref](), meta); err != nil {
		m.logger.Error("failed to write access audit log", slog.String("error", err.Error()))
	}
}
