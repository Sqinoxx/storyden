package account_deletion

import (
	"context"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/account/account_querier"
	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/internal/infrastructure/pubsub"
)

type Service interface {
	Delete(ctx context.Context, id account.AccountID) error
}

func Build() fx.Option {
	return fx.Provide(New)
}

type service struct {
	account_querier *account_querier.Querier
	account_writer  *account_writer.Writer
	bus             *pubsub.Bus
}

func New(
	account_querier *account_querier.Querier,
	account_writer *account_writer.Writer,
	bus *pubsub.Bus,
) Service {
	return &service{
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

	err = s.account_writer.Delete(ctx, id)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	return nil
}
