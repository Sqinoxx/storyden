package access

import (
	"time"

	"github.com/Southclaws/dt"
	"github.com/Southclaws/opt"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/post/category"
	"github.com/Southclaws/storyden/app/resources/profile"
	"github.com/Southclaws/storyden/internal/ent"
)

//go:generate go run github.com/Southclaws/enumerator

type grantStatusEnum string

const (
	grantStatusRequested grantStatusEnum = "requested"
	grantStatusActive    grantStatusEnum = "active"
	grantStatusRejected  grantStatusEnum = "rejected"
	grantStatusRevoked   grantStatusEnum = "revoked"
)

type PassID xid.ID

func (i PassID) String() string { return xid.ID(i).String() }

type GrantID xid.ID

func (i GrantID) String() string { return xid.ID(i).String() }

type PassRef struct {
	ID         PassID
	Name       string
	PriceLabel string
	Colour     string
}

type Pass struct {
	PassRef
	Description  string
	PaymentInfo  string
	Sort         int
	Requestable  bool
	Categories   []category.CategoryID
	DriveFolders []xid.ID
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Passes []*Pass

type Grant struct {
	ID            GrantID
	AccountID     account.AccountID
	PassID        PassID
	Status        GrantStatus
	ReferenceCode string
	ExpiresAt     opt.Optional[time.Time]
	RequestedAt   opt.Optional[time.Time]
	DecidedBy     opt.Optional[account.AccountID]
	DecidedAt     opt.Optional[time.Time]
	Note          string
	Reason        string
	CreatedAt     time.Time
	UpdatedAt     time.Time

	Account         opt.Optional[profile.Ref]
	Pass            opt.Optional[PassRef]
	DecidedByMember opt.Optional[profile.Ref]
}

// DisplayStatus is what members and managers see: an active grant past its
// expiry reads as expired even though nothing was written when it lapsed.
func (g *Grant) DisplayStatus(now time.Time) string {
	if g.Status == GrantStatusActive && !g.IsEffective(now) {
		return "expired"
	}
	return g.Status.String()
}

func (g *Grant) IsEffective(now time.Time) bool {
	if g.Status != GrantStatusActive {
		return false
	}

	if exp, ok := g.ExpiresAt.Get(); ok && !now.Before(exp) {
		return false
	}

	return true
}

// Lock describes why a piece of content is withheld from the requester: it
// sits under a category covered by these passes and they hold none of them.
type Lock struct {
	Passes []PassRef
}

func MapPassRef(in *ent.AccessPass) PassRef {
	return PassRef{
		ID:         PassID(in.ID),
		Name:       in.Name,
		PriceLabel: in.PriceLabel,
		Colour:     in.Colour,
	}
}

func MapPass(in *ent.AccessPass) *Pass {
	return &Pass{
		PassRef:     MapPassRef(in),
		Description: in.Description,
		PaymentInfo: in.PaymentInfo,
		Sort:        in.Sort,
		Requestable: in.Requestable,
		Categories: dt.Map(in.Edges.Categories, func(c *ent.Category) category.CategoryID {
			return category.CategoryID(c.ID)
		}),
		DriveFolders: dt.Map(in.Edges.DriveFolders, func(f *ent.DriveFolder) xid.ID {
			return f.ID
		}),
		CreatedAt: in.CreatedAt,
		UpdatedAt: in.UpdatedAt,
	}
}

func MapGrant(in *ent.AccessGrant) (*Grant, error) {
	status, err := NewGrantStatus(in.Status.String())
	if err != nil {
		return nil, err
	}

	acc, err := opt.MapErr(opt.NewPtr(in.Edges.Account), func(a ent.Account) (profile.Ref, error) {
		r, err := profile.MapRef(&a)
		if err != nil {
			return profile.Ref{}, err
		}
		return *r, nil
	})
	if err != nil {
		return nil, err
	}

	pass := opt.Map(opt.NewPtr(in.Edges.Pass), func(p ent.AccessPass) PassRef { return MapPassRef(&p) })

	return &Grant{
		Account:       acc,
		Pass:          pass,
		ID:            GrantID(in.ID),
		AccountID:     account.AccountID(in.AccountID),
		PassID:        PassID(in.PassID),
		Status:        status,
		ReferenceCode: in.ReferenceCode,
		ExpiresAt:     opt.NewPtr(in.ExpiresAt),
		RequestedAt:   opt.NewPtr(in.RequestedAt),
		DecidedBy:     opt.Map(opt.NewPtr(in.DecidedBy), func(id xid.ID) account.AccountID { return account.AccountID(id) }),
		DecidedAt:     opt.NewPtr(in.DecidedAt),
		Note:          in.Note,
		Reason:        in.Reason,
		CreatedAt:     in.CreatedAt,
		UpdatedAt:     in.UpdatedAt,
	}, nil
}
