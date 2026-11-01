package access_repo

import (
	"context"
	"crypto/rand"
	"math/big"
	"strings"
	"time"
	"unicode"

	"github.com/Southclaws/dt"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/Southclaws/opt"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/access"
	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/post/category"
	"github.com/Southclaws/storyden/app/resources/profile"
	"github.com/Southclaws/storyden/internal/ent"
	"github.com/Southclaws/storyden/internal/ent/accessgrant"
	"github.com/Southclaws/storyden/internal/ent/accesspass"
	ent_account "github.com/Southclaws/storyden/internal/ent/account"
	ent_asset "github.com/Southclaws/storyden/internal/ent/asset"
	ent_category "github.com/Southclaws/storyden/internal/ent/category"
	ent_post "github.com/Southclaws/storyden/internal/ent/post"
)

var ErrNotFound = fault.New("access pass not found", ftag.With(ftag.NotFound))

type Repository struct {
	db *ent.Client
}

func New(db *ent.Client) *Repository {
	return &Repository{db: db}
}

type PassOption func(*ent.AccessPassMutation)

func WithName(v string) PassOption {
	return func(m *ent.AccessPassMutation) { m.SetName(v) }
}

func WithDescription(v string) PassOption {
	return func(m *ent.AccessPassMutation) { m.SetDescription(v) }
}

func WithPriceLabel(v string) PassOption {
	return func(m *ent.AccessPassMutation) { m.SetPriceLabel(v) }
}

func WithPaymentInfo(v string) PassOption {
	return func(m *ent.AccessPassMutation) { m.SetPaymentInfo(v) }
}

func WithColour(v string) PassOption {
	return func(m *ent.AccessPassMutation) { m.SetColour(v) }
}

func WithSort(v int) PassOption {
	return func(m *ent.AccessPassMutation) { m.SetSort(v) }
}

func WithRequestable(v bool) PassOption {
	return func(m *ent.AccessPassMutation) { m.SetRequestable(v) }
}

func WithCategories(ids []category.CategoryID) PassOption {
	return func(m *ent.AccessPassMutation) {
		m.ClearCategories()
		m.AddCategoryIDs(dt.Map(ids, func(id category.CategoryID) xid.ID { return xid.ID(id) })...)
	}
}

func (r *Repository) CreatePass(ctx context.Context, name string, opts ...PassOption) (*access.Pass, error) {
	create := r.db.AccessPass.Create().SetName(name)
	mutation := create.Mutation()
	for _, fn := range opts {
		fn(mutation)
	}

	p, err := create.Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.InvalidArgument))
		}
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return r.GetPass(ctx, access.PassID(p.ID))
}

func (r *Repository) UpdatePass(ctx context.Context, id access.PassID, opts ...PassOption) (*access.Pass, error) {
	update := r.db.AccessPass.UpdateOneID(xid.ID(id))
	mutation := update.Mutation()
	for _, fn := range opts {
		fn(mutation)
	}

	if err := update.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return nil, fault.Wrap(ErrNotFound, fctx.With(ctx))
		}
		if ent.IsConstraintError(err) {
			return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.InvalidArgument))
		}
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return r.GetPass(ctx, id)
}

func (r *Repository) DeletePass(ctx context.Context, id access.PassID) error {
	if err := r.db.AccessPass.DeleteOneID(xid.ID(id)).Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return fault.Wrap(ErrNotFound, fctx.With(ctx))
		}
		return fault.Wrap(err, fctx.With(ctx))
	}
	return nil
}

func (r *Repository) GetPass(ctx context.Context, id access.PassID) (*access.Pass, error) {
	p, err := r.db.AccessPass.Query().
		Where(accesspass.ID(xid.ID(id))).
		WithCategories(func(cq *ent.CategoryQuery) { cq.Select(ent_category.FieldID) }).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fault.Wrap(ErrNotFound, fctx.With(ctx))
		}
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return access.MapPass(p), nil
}

func (r *Repository) ListPasses(ctx context.Context) (access.Passes, error) {
	ps, err := r.db.AccessPass.Query().
		WithCategories(func(cq *ent.CategoryQuery) { cq.Select(ent_category.FieldID) }).
		Order(ent.Asc(accesspass.FieldSort), ent.Asc(accesspass.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return dt.Map(ps, access.MapPass), nil
}

type GrantOptions struct {
	ExpiresAt opt.Optional[time.Time]
	DecidedBy opt.Optional[account.AccountID]
	Note      opt.Optional[string]
}

// SetGrant moves the account's single grant row for a pass to the given
// status, creating it first if it does not exist yet. The reference code is
// assigned once and kept across every later transition.
func (r *Repository) SetGrant(ctx context.Context, accountID account.AccountID, passID access.PassID, status access.GrantStatus, opts GrantOptions) (*access.Grant, error) {
	existing, err := r.db.AccessGrant.Query().
		Where(
			accessgrant.AccountID(xid.ID(accountID)),
			accessgrant.PassID(xid.ID(passID)),
		).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	now := time.Now()
	entStatus := accessgrant.Status(status.String())

	if existing != nil {
		update := existing.Update().SetStatus(entStatus)
		applyGrantTransition(update.Mutation(), status, now, opts)
		g, err := update.Save(ctx)
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}
		return access.MapGrant(g)
	}

	pass, err := r.db.AccessPass.Get(ctx, xid.ID(passID))
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fault.Wrap(ErrNotFound, fctx.With(ctx))
		}
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	for attempt := 0; ; attempt++ {
		code, err := newReferenceCode(pass.Name)
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}

		create := r.db.AccessGrant.Create().
			SetAccountID(xid.ID(accountID)).
			SetPassID(xid.ID(passID)).
			SetStatus(entStatus).
			SetReferenceCode(code)
		applyGrantTransition(create.Mutation(), status, now, opts)

		g, err := create.Save(ctx)
		if err != nil {
			if ent.IsConstraintError(err) && attempt < 5 {
				continue
			}
			return nil, fault.Wrap(err, fctx.With(ctx))
		}

		return access.MapGrant(g)
	}
}

func applyGrantTransition(m *ent.AccessGrantMutation, status access.GrantStatus, now time.Time, opts GrantOptions) {
	switch status {
	case access.GrantStatusRequested:
		m.SetRequestedAt(now)
		m.ClearDecidedAt()
		m.ClearDecidedBy()
		m.ClearExpiresAt()

	case access.GrantStatusActive:
		m.SetDecidedAt(now)
		m.ClearExpiryNotifiedAt()
		if exp, ok := opts.ExpiresAt.Get(); ok {
			m.SetExpiresAt(exp)
		} else {
			m.ClearExpiresAt()
		}

	default:
		m.SetDecidedAt(now)
	}

	if by, ok := opts.DecidedBy.Get(); ok {
		m.SetDecidedBy(xid.ID(by))
	}

	if note, ok := opts.Note.Get(); ok {
		m.SetNote(note)
	}
}

func (r *Repository) GetGrant(ctx context.Context, accountID account.AccountID, passID access.PassID) (*access.Grant, error) {
	g, err := r.db.AccessGrant.Query().
		Where(
			accessgrant.AccountID(xid.ID(accountID)),
			accessgrant.PassID(xid.ID(passID)),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.NotFound), fmsg.With("grant not found"))
		}
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return access.MapGrant(g)
}

func (r *Repository) ListAccountGrants(ctx context.Context, accountID account.AccountID) ([]*access.Grant, error) {
	gs, err := r.db.AccessGrant.Query().
		Where(accessgrant.AccountID(xid.ID(accountID))).
		All(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return dt.MapErr(gs, access.MapGrant)
}

type GrantFilter struct {
	Status    opt.Optional[string]
	PassID    opt.Optional[access.PassID]
	AccountID opt.Optional[account.AccountID]
}

func withMember(aq *ent.AccountQuery) {
	aq.WithAccountRoles(func(arq *ent.AccountRolesQuery) {
		arq.WithRole()
	})
}

func (r *Repository) ListGrants(ctx context.Context, filter GrantFilter, now time.Time) ([]*access.Grant, error) {
	q := r.db.AccessGrant.Query().
		WithAccount(withMember).
		WithPass().
		Order(ent.Desc(accessgrant.FieldUpdatedAt))

	if status, ok := filter.Status.Get(); ok {
		switch status {
		case "expired":
			q.Where(accessgrant.StatusEQ(accessgrant.StatusActive), accessgrant.ExpiresAtLTE(now))
		case "active":
			q.Where(accessgrant.StatusEQ(accessgrant.StatusActive), accessgrant.Or(
				accessgrant.ExpiresAtIsNil(),
				accessgrant.ExpiresAtGT(now),
			))
		default:
			q.Where(accessgrant.StatusEQ(accessgrant.Status(status)))
		}
	}
	filter.PassID.Call(func(id access.PassID) { q.Where(accessgrant.PassID(xid.ID(id))) })
	filter.AccountID.Call(func(id account.AccountID) { q.Where(accessgrant.AccountID(xid.ID(id))) })

	gs, err := q.All(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return r.withDeciders(ctx, gs)
}

func (r *Repository) GetGrantDetailed(ctx context.Context, accountID account.AccountID, passID access.PassID) (*access.Grant, error) {
	g, err := r.db.AccessGrant.Query().
		Where(
			accessgrant.AccountID(xid.ID(accountID)),
			accessgrant.PassID(xid.ID(passID)),
		).
		WithAccount(withMember).
		WithPass().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.NotFound), fmsg.With("grant not found"))
		}
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	gs, err := r.withDeciders(ctx, []*ent.AccessGrant{g})
	if err != nil {
		return nil, err
	}

	return gs[0], nil
}

// withDeciders maps grants and attaches the profile of whoever decided each
// one. decided_by is a plain column rather than an edge so deleting a manager
// does not wipe the history of what they granted.
func (r *Repository) withDeciders(ctx context.Context, gs []*ent.AccessGrant) ([]*access.Grant, error) {
	ids := []xid.ID{}
	for _, g := range gs {
		if g.DecidedBy != nil {
			ids = append(ids, *g.DecidedBy)
		}
	}

	deciders := map[xid.ID]profile.Ref{}
	if len(ids) > 0 {
		accs, err := r.db.Account.Query().
			Where(ent_account.IDIn(ids...)).
			WithAccountRoles(func(arq *ent.AccountRolesQuery) { arq.WithRole() }).
			All(ctx)
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}
		for _, a := range accs {
			ref, err := profile.MapRef(a)
			if err != nil {
				return nil, fault.Wrap(err, fctx.With(ctx))
			}
			deciders[a.ID] = *ref
		}
	}

	out := make([]*access.Grant, 0, len(gs))
	for _, g := range gs {
		mapped, err := access.MapGrant(g)
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}
		if g.DecidedBy != nil {
			if ref, ok := deciders[*g.DecidedBy]; ok {
				mapped.DecidedByMember = opt.New(ref)
			}
		}
		out = append(out, mapped)
	}

	return out, nil
}

func (r *Repository) DeleteGrant(ctx context.Context, id access.GrantID) error {
	if err := r.db.AccessGrant.DeleteOneID(xid.ID(id)).Exec(ctx); err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}
	return nil
}

// ListExpiringUnnotified finds active grants that lapse within the window and
// whose holder has not been warned yet.
func (r *Repository) ListExpiringUnnotified(ctx context.Context, now, until time.Time) ([]*access.Grant, error) {
	gs, err := r.db.AccessGrant.Query().
		Where(
			accessgrant.StatusEQ(accessgrant.StatusActive),
			accessgrant.ExpiresAtGT(now),
			accessgrant.ExpiresAtLTE(until),
			accessgrant.ExpiryNotifiedAtIsNil(),
		).
		WithPass().
		All(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return dt.MapErr(gs, access.MapGrant)
}

func (r *Repository) MarkExpiryNotified(ctx context.Context, id access.GrantID, at time.Time) error {
	if err := r.db.AccessGrant.UpdateOneID(xid.ID(id)).SetExpiryNotifiedAt(at).Exec(ctx); err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}
	return nil
}

// HeldPassIDs returns the passes the account currently holds an effective
// grant for. Expiry is evaluated here so no background job is needed.
func (r *Repository) HeldPassIDs(ctx context.Context, accountID account.AccountID, now time.Time) (map[access.PassID]struct{}, error) {
	gs, err := r.db.AccessGrant.Query().
		Where(
			accessgrant.AccountID(xid.ID(accountID)),
			accessgrant.StatusEQ(accessgrant.StatusActive),
			accessgrant.Or(
				accessgrant.ExpiresAtIsNil(),
				accessgrant.ExpiresAtGT(now),
			),
		).
		Select(accessgrant.FieldPassID).
		All(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	held := make(map[access.PassID]struct{}, len(gs))
	for _, g := range gs {
		held[access.PassID(g.PassID)] = struct{}{}
	}

	return held, nil
}

// CategoryLocks maps every category covered by at least one pass, directly or
// through an ancestor, to the passes that unlock it. An installation without
// passes costs a single query that returns nothing.
func (r *Repository) CategoryLocks(ctx context.Context) (map[category.CategoryID][]access.PassRef, error) {
	passes, err := r.db.AccessPass.Query().
		Where(accesspass.HasCategories()).
		WithCategories(func(cq *ent.CategoryQuery) { cq.Select(ent_category.FieldID) }).
		All(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if len(passes) == 0 {
		return map[category.CategoryID][]access.PassRef{}, nil
	}

	direct := map[xid.ID][]access.PassRef{}
	for _, p := range passes {
		ref := access.MapPassRef(p)
		for _, c := range p.Edges.Categories {
			direct[c.ID] = append(direct[c.ID], ref)
		}
	}

	cats, err := r.db.Category.Query().
		Select(ent_category.FieldID, ent_category.FieldParentCategoryID).
		All(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	parents := make(map[xid.ID]xid.ID, len(cats))
	for _, c := range cats {
		parents[c.ID] = c.ParentCategoryID
	}

	return ResolveCategoryLocks(parents, direct), nil
}

// ResolveCategoryLocks walks each category up to its root and collects the
// passes attached anywhere along the way.
func ResolveCategoryLocks(parents map[xid.ID]xid.ID, direct map[xid.ID][]access.PassRef) map[category.CategoryID][]access.PassRef {
	out := map[category.CategoryID][]access.PassRef{}

	for id := range parents {
		var refs []access.PassRef
		seen := map[access.PassID]struct{}{}

		cursor := id
		for depth := 0; !cursor.IsNil() && depth <= category.MaxCategoryDepth+1; depth++ {
			for _, ref := range direct[cursor] {
				if _, dup := seen[ref.ID]; dup {
					continue
				}
				seen[ref.ID] = struct{}{}
				refs = append(refs, ref)
			}
			cursor = parents[cursor]
		}

		if len(refs) > 0 {
			out[category.CategoryID(id)] = refs
		}
	}

	return out
}

// PostOwnership is what an access decision needs to know about a post: the
// thread author, who may always read their own thread, and its category.
type PostOwnership struct {
	ThreadAuthor account.AccountID
	CategoryID   xid.ID
}

// PostThreads resolves posts, threads or replies alike, to the thread they
// belong to. Unknown IDs are absent from the result.
func (r *Repository) PostThreads(ctx context.Context, ids ...xid.ID) (map[xid.ID]PostOwnership, error) {
	posts, err := r.db.Post.Query().
		Where(ent_post.IDIn(ids...)).
		Select(ent_post.FieldID, ent_post.FieldRootPostID, ent_post.FieldCategoryID, ent_post.FieldAccountPosts).
		All(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	rootIDs := []xid.ID{}
	for _, p := range posts {
		if p.RootPostID != nil && !p.RootPostID.IsNil() {
			rootIDs = append(rootIDs, *p.RootPostID)
		}
	}

	roots := map[xid.ID]*ent.Post{}
	if len(rootIDs) > 0 {
		rs, err := r.db.Post.Query().
			Where(ent_post.IDIn(rootIDs...)).
			Select(ent_post.FieldID, ent_post.FieldCategoryID, ent_post.FieldAccountPosts).
			All(ctx)
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}
		for _, root := range rs {
			roots[root.ID] = root
		}
	}

	out := make(map[xid.ID]PostOwnership, len(posts))
	for _, p := range posts {
		thread := p
		if p.RootPostID != nil && !p.RootPostID.IsNil() {
			root, ok := roots[*p.RootPostID]
			if !ok {
				continue
			}
			thread = root
		}

		out[p.ID] = PostOwnership{
			ThreadAuthor: account.AccountID(thread.AccountPosts),
			CategoryID:   thread.CategoryID,
		}
	}

	return out, nil
}

// AssetPosts lists the posts an asset is attached to, either directly or
// through the original it was derived from, such as a resized image.
func (r *Repository) AssetPosts(ctx context.Context, assetID xid.ID) ([]xid.ID, error) {
	ids, err := r.db.Post.Query().
		Where(ent_post.HasAssetsWith(ent_asset.Or(
			ent_asset.ID(assetID),
			ent_asset.HasAssetsWith(ent_asset.ID(assetID)),
		))).
		IDs(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return ids, nil
}

const referenceAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func newReferenceCode(passName string) (string, error) {
	prefix := strings.Builder{}
	for _, r := range passName {
		if prefix.Len() >= 4 {
			break
		}
		if r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
			prefix.WriteRune(unicode.ToUpper(r))
			continue
		}
		if prefix.Len() > 0 {
			break
		}
	}
	if prefix.Len() == 0 {
		prefix.WriteString("SD")
	}

	suffix := make([]byte, 4)
	for i := range suffix {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(referenceAlphabet))))
		if err != nil {
			return "", err
		}
		suffix[i] = referenceAlphabet[n.Int64()]
	}

	return prefix.String() + "-" + string(suffix), nil
}
