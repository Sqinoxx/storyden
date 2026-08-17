package invitation

import (
	"time"

	"github.com/Southclaws/opt"
	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/internal/ent"
	"github.com/rs/xid"
)

type Invitation struct {
	ID        xid.ID
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt opt.Optional[time.Time]
	Message   opt.Optional[string]
	Creator   account.Account
	MaxUses   opt.Optional[int]
	ExpiresAt opt.Optional[time.Time]
	Uses      int
}

func Map(in *ent.Invitation) (*Invitation, error) {
	creatorEdge, err := in.Edges.CreatorOrErr()
	if err != nil {
		return nil, err
	}

	acc, err := account.MapRef(creatorEdge)
	if err != nil {
		return nil, err
	}

	return &Invitation{
		ID:        in.ID,
		CreatedAt: in.CreatedAt,
		UpdatedAt: in.UpdatedAt,
		DeletedAt: opt.NewPtr(in.DeletedAt),
		Message:   opt.NewPtr(in.Message),
		Creator:   *acc,
		MaxUses:   opt.NewPtr(in.MaxUses),
		ExpiresAt: opt.NewPtr(in.ExpiresAt),
		Uses:      len(in.Edges.Invited),
	}, nil
}
