package bug_report

import (
	"time"

	"github.com/Southclaws/opt"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/internal/ent"
)

type ID xid.ID

func (i ID) String() string { return xid.ID(i).String() }

type BugReport struct {
	ID          ID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Author      account.Account
	Category    Category
	Description string
	PageURL     opt.Optional[string]
	UserAgent   opt.Optional[string]
	Status      Status
	Screenshot  opt.Optional[asset.Asset]
}

func Map(r *ent.BugReport) (*BugReport, error) {
	authorEdge, err := r.Edges.AuthorOrErr()
	if err != nil {
		return nil, err
	}

	author, err := account.MapRef(authorEdge)
	if err != nil {
		return nil, err
	}

	category, err := NewCategory(r.Category)
	if err != nil {
		return nil, err
	}

	status, err := NewStatus(r.Status)
	if err != nil {
		return nil, err
	}

	screenshot := opt.NewPtrMap(r.Edges.Screenshot, func(a ent.Asset) asset.Asset { return *asset.Map(&a) })

	return &BugReport{
		ID:          ID(r.ID),
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
		Author:      *author,
		Category:    category,
		Description: r.Description,
		PageURL:     opt.NewPtr(r.PageURL),
		UserAgent:   opt.NewPtr(r.UserAgent),
		Status:      status,
		Screenshot:  screenshot,
	}, nil
}
