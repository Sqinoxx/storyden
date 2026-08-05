package account_deletion

import (
	"context"
	"fmt"
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
	ent_session "github.com/Southclaws/storyden/internal/ent/session"
	"github.com/Southclaws/storyden/internal/infrastructure/pubsub"
)

type Service interface {
	Delete(ctx context.Context, id account.AccountID) error
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

	xidID := xid.ID(id)

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
	_, err = s.account_writer.Update(ctx, id,
		account_writer.SetHandle(newHandle),
		account_writer.SetName("Deleted User"),
		account_writer.SetBio(""),
		account_writer.SetSignature(""),
		account_writer.SetLinks([]account.ExternalLink{}),
		account_writer.SetMetadata(map[string]any{}),
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
