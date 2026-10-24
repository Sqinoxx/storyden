package bug_report

import (
	"context"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/Southclaws/opt"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/account/account_querier"
	"github.com/Southclaws/storyden/app/resources/account/notification"
	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/app/resources/bug_report"
	"github.com/Southclaws/storyden/app/resources/bug_report/bug_report_querier"
	"github.com/Southclaws/storyden/app/resources/bug_report/bug_report_writer"
	"github.com/Southclaws/storyden/app/resources/pagination"
	"github.com/Southclaws/storyden/app/resources/rbac"
	"github.com/Southclaws/storyden/app/services/authentication/session"
	"github.com/Southclaws/storyden/app/services/notification/notify"
)

const (
	maxDescriptionLength = 5000
	maxPageURLLength     = 2000
	maxUserAgentLength   = 500
)

type Manager struct {
	logger         *slog.Logger
	querier        *bug_report_querier.Querier
	writer         *bug_report_writer.Writer
	accountQuerier *account_querier.Querier
	notifier       *notify.Notifier
}

func New(
	logger *slog.Logger,
	querier *bug_report_querier.Querier,
	writer *bug_report_writer.Writer,
	accountQuerier *account_querier.Querier,
	notifier *notify.Notifier,
) *Manager {
	return &Manager{
		logger:         logger,
		querier:        querier,
		writer:         writer,
		accountQuerier: accountQuerier,
		notifier:       notifier,
	}
}

type SubmitParams struct {
	Category    bug_report.Category
	Description string
	PageURL     opt.Optional[string]
	UserAgent   opt.Optional[string]
	Screenshot  opt.Optional[asset.AssetID]
}

func (m *Manager) Submit(ctx context.Context, p SubmitParams) (*bug_report.BugReport, error) {
	accountID, err := session.GetAccountID(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	description := strings.TrimSpace(p.Description)
	if description == "" {
		return nil, fault.New("description is required", fctx.With(ctx), ftag.With(ftag.InvalidArgument), fmsg.WithDesc("empty", "Please describe the problem."))
	}
	if utf8.RuneCountInString(description) > maxDescriptionLength {
		return nil, fault.New("description too long", fctx.With(ctx), ftag.With(ftag.InvalidArgument), fmsg.WithDesc("too long", "The description is too long."))
	}

	rep, err := m.writer.Create(ctx, accountID, bug_report_writer.CreateParams{
		Category:    p.Category,
		Description: description,
		PageURL:     truncate(p.PageURL, maxPageURLLength),
		UserAgent:   truncate(p.UserAgent, maxUserAgentLength),
		Screenshot:  p.Screenshot,
	})
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if err := m.notifyAdmins(ctx, accountID); err != nil {
		m.logger.Error("failed to notify admins of bug report", slog.String("error", err.Error()))
	}

	return rep, nil
}

func (m *Manager) List(ctx context.Context, page pagination.Parameters, status opt.Optional[bug_report.Status]) (pagination.Result[*bug_report.BugReport], error) {
	return m.querier.List(ctx, page, status)
}

func (m *Manager) UpdateStatus(ctx context.Context, id bug_report.ID, status bug_report.Status) (*bug_report.BugReport, error) {
	return m.writer.UpdateStatus(ctx, id, status)
}

func (m *Manager) Delete(ctx context.Context, id bug_report.ID) error {
	return m.writer.Delete(ctx, id)
}

func (m *Manager) notifyAdmins(ctx context.Context, source account.AccountID) error {
	admins, err := m.accountQuerier.ListByHeldPermission(ctx, rbac.PermissionAdministrator)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	for _, admin := range admins {
		if admin.ID == source {
			continue
		}
		if err := m.notifier.Send(ctx, admin.ID, opt.New(source), notification.EventBugReportSubmitted, nil); err != nil {
			return fault.Wrap(err, fctx.With(ctx))
		}
	}

	return nil
}

func truncate(in opt.Optional[string], max int) opt.Optional[string] {
	v, ok := in.Get()
	if !ok {
		return in
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return opt.NewEmpty[string]()
	}
	if r := []rune(v); len(r) > max {
		v = string(r[:max])
	}
	return opt.New(v)
}
