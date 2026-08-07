package account_deletion

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/Southclaws/opt"
	"github.com/rs/xid"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/account/account_querier"
	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/internal/ent"
	ent_account "github.com/Southclaws/storyden/internal/ent/account"
	ent_accountfollow "github.com/Southclaws/storyden/internal/ent/accountfollow"
	ent_accountroles "github.com/Southclaws/storyden/internal/ent/accountroles"
	ent_authentication "github.com/Southclaws/storyden/internal/ent/authentication"
	ent_email "github.com/Southclaws/storyden/internal/ent/email"
	ent_notification "github.com/Southclaws/storyden/internal/ent/notification"
	ent_oauthrefreshtoken "github.com/Southclaws/storyden/internal/ent/oauthrefreshtoken"
	ent_oauthremoteconnection "github.com/Southclaws/storyden/internal/ent/oauthremoteconnection"
	ent_post "github.com/Southclaws/storyden/internal/ent/post"
	ent_session "github.com/Southclaws/storyden/internal/ent/session"
	"github.com/Southclaws/storyden/internal/infrastructure/pubsub"
)

// deletedPlaceholderHandle identifies the single system account that authored
// content is reassigned to when an account is permanently deleted by an admin.
const deletedPlaceholderHandle = "deleted-user"

type Service interface {
	Delete(ctx context.Context, id account.AccountID) error
	SelfDelete(ctx context.Context, id account.AccountID) error
}

func Build() fx.Option {
	return fx.Provide(New)
}

type service struct {
	db              *ent.Client
	account_querier *account_querier.Querier
	account_writer  *account_writer.Writer
	bus             *pubsub.Bus
}

func New(
	db *ent.Client,
	account_querier *account_querier.Querier,
	account_writer *account_writer.Writer,
	bus *pubsub.Bus,
) Service {
	return &service{
		db:              db,
		account_querier: account_querier,
		account_writer:  account_writer,
		bus:             bus,
	}
}

func (s *service) Delete(ctx context.Context, id account.AccountID) error {
	acc, err := s.account_querier.GetByID(ctx, id)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	// Precondition: account MUST be suspended before deletion
	if !acc.DeletedAt.Ok() {
		return fault.Wrap(
			fault.New("account must be suspended before deletion"),
			ftag.With(ftag.InvalidArgument),
			fctx.With(ctx),
			fmsg.WithDesc("precondition failed", "account must be suspended before deletion"),
		)
	}

	return s.hardDelete(ctx, acc)
}

func (s *service) SelfDelete(ctx context.Context, id account.AccountID) error {
	acc, err := s.account_querier.GetByID(ctx, id)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	if acc.Admin {
		return fault.Wrap(
			fault.New("administrators cannot delete their own account"),
			ftag.With(ftag.PermissionDenied),
			fctx.With(ctx),
			fmsg.WithDesc("permission denied", "Administratoren können ihr eigenes Konto nicht löschen."),
		)
	}


	return s.anonymizeAndDelete(ctx, acc)
}

func (s *service) anonymizeAndDelete(ctx context.Context, acc *account.AccountWithEdges) error {
	id := acc.ID
	xidID := xid.ID(id)

	meta := map[string]any{}
	if acc.Metadata != nil {
		for k, v := range acc.Metadata {
			meta[k] = v
		}
	}
	if acc.Name != "" && acc.Name != "Deleted User" {
		meta["original_name"] = acc.Name
	}
	if acc.Handle != "" && !strings.HasPrefix(acc.Handle, "deleted-") {
		meta["original_handle"] = acc.Handle
	}
	if len(acc.EmailAddresses) > 0 && acc.EmailAddresses[0] != nil {
		meta["original_email"] = acc.EmailAddresses[0].Email.Address
	}



	// Delete sensitive sub-records / credentials / sessions
	_, _ = s.db.Session.Delete().Where(ent_session.HasAccountWith(ent_account.IDEQ(xidID))).Exec(ctx)
	_, _ = s.db.Email.Delete().Where(ent_email.HasAccountWith(ent_account.IDEQ(xidID))).Exec(ctx)
	_, _ = s.db.Authentication.Delete().Where(ent_authentication.HasAccountWith(ent_account.IDEQ(xidID))).Exec(ctx)
	_, _ = s.db.OAuthRemoteConnection.Delete().Where(ent_oauthremoteconnection.HasAccountWith(ent_account.IDEQ(xidID))).Exec(ctx)
	_, _ = s.db.OAuthRefreshToken.Delete().Where(ent_oauthrefreshtoken.HasAccountWith(ent_account.IDEQ(xidID))).Exec(ctx)
	_, _ = s.db.AccountRoles.Delete().Where(ent_accountroles.HasAccountWith(ent_account.IDEQ(xidID))).Exec(ctx)
	_, _ = s.db.AccountFollow.Delete().Where(ent_accountfollow.Or(ent_accountfollow.HasFollowerWith(ent_account.IDEQ(xidID)), ent_accountfollow.HasFollowingWith(ent_account.IDEQ(xidID)))).Exec(ctx)
	_, _ = s.db.Notification.Delete().Where(ent_notification.Or(ent_notification.HasOwnerWith(ent_account.IDEQ(xidID)), ent_notification.HasSourceWith(ent_account.IDEQ(xidID)))).Exec(ctx)

	// Anonymize Account: change handle and name to "Deleted User" so posts are retained
	newHandle := fmt.Sprintf("deleted-%s", id.String())
	_, err := s.account_writer.Update(ctx, id,
		account_writer.SetHandle(newHandle),
		account_writer.SetName("Deleted User"),
		account_writer.SetBio(""),
		account_writer.SetSignature(""),
		account_writer.SetLinks([]account.ExternalLink{}),
		account_writer.SetMetadata(meta),
		account_writer.SetVerifiedStatus(account.VerifiedStatusNone),
		account_writer.SetAdmin(false),
		account_writer.SetDeleted(opt.New(time.Now())),
	)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	s.account_writer.InvalidateCache(ctx, id)

	return nil
}

// hardDelete removes the account row entirely. Authored posts/threads/replies
// are reassigned to a shared "Deleted User" placeholder account first, since
// the required Post.author edge otherwise blocks deletion and would delete
// content along with the account via cascade.
func (s *service) hardDelete(ctx context.Context, acc *account.AccountWithEdges) error {
	id := acc.ID
	xidID := xid.ID(id)

	placeholderID, err := s.getOrCreateDeletedPlaceholder(ctx)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	if xidID == placeholderID {
		return fault.Wrap(
			fault.New("cannot delete the system placeholder account"),
			ftag.With(ftag.InvalidArgument),
			fctx.With(ctx),
			fmsg.WithDesc("invalid action", "Dieses Systemkonto kann nicht gelöscht werden."),
		)
	}

	_, err = s.db.Post.Update().
		Where(ent_post.HasAuthorWith(ent_account.IDEQ(xidID))).
		SetAccountPosts(placeholderID).
		Save(ctx)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	if err := s.db.Account.DeleteOneID(xidID).Exec(ctx); err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	s.account_writer.InvalidateCache(ctx, id)

	return nil
}

func (s *service) getOrCreateDeletedPlaceholder(ctx context.Context) (xid.ID, error) {
	existing, err := s.db.Account.Query().
		Where(ent_account.HandleEQ(deletedPlaceholderHandle)).
		Only(ctx)
	if err == nil {
		return existing.ID, nil
	}
	if !ent.IsNotFound(err) {
		return xid.ID{}, fault.Wrap(err, fctx.With(ctx))
	}

	created, err := s.db.Account.Create().
		SetHandle(deletedPlaceholderHandle).
		SetName("Deleted User").
		SetDeletedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return xid.ID{}, fault.Wrap(err, fctx.With(ctx))
	}

	return created.ID, nil
}

