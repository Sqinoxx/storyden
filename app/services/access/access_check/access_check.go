// Package access_check decides which categories a requester may read in full
// and withholds the content of threads under categories they have not paid
// the student council for. Titles and metadata stay visible as a preview.
package access_check

import (
	"context"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/Southclaws/opt"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/access"
	"github.com/Southclaws/storyden/app/resources/access/access_repo"
	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/datagraph"
	"github.com/Southclaws/storyden/app/resources/link/link_ref"
	"github.com/Southclaws/storyden/app/resources/pagination"
	"github.com/Southclaws/storyden/app/resources/post/category"
	"github.com/Southclaws/storyden/app/resources/post/reply"
	"github.com/Southclaws/storyden/app/resources/post/thread"
	"github.com/Southclaws/storyden/app/resources/rbac"
	"github.com/Southclaws/storyden/app/services/authentication/session"
)

var ErrLocked = fault.New("content requires an access pass", ftag.With(ftag.PermissionDenied))

type Checker struct {
	repo *access_repo.Repository
}

func New(repo *access_repo.Repository) *Checker {
	return &Checker{repo: repo}
}

type Decision struct {
	bypass  bool
	account opt.Optional[account.AccountID]
	locks   map[category.CategoryID][]access.PassRef
	held    map[access.PassID]struct{}
}

func (c *Checker) Resolve(ctx context.Context) (*Decision, error) {
	locks, err := c.repo.CategoryLocks(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if len(locks) == 0 || bypasses(ctx) {
		return &Decision{bypass: true}, nil
	}

	accountID, held, err := c.held(ctx)
	if err != nil {
		return nil, err
	}

	return &Decision{
		account: accountID,
		locks:   locks,
		held:    held,
	}, nil
}

// bypasses covers internal jobs, which carry no session and are trusted, and
// staff who must be able to moderate or manage locked content.
func bypasses(ctx context.Context) bool {
	if !session.HasContext(ctx) {
		return true
	}
	return session.Authorise(ctx, nil, rbac.PermissionManageAccess, rbac.PermissionManagePosts) == nil
}

func (c *Checker) held(ctx context.Context) (opt.Optional[account.AccountID], map[access.PassID]struct{}, error) {
	accountID := session.GetOptAccountID(ctx)

	id, ok := accountID.Get()
	if !ok {
		return accountID, map[access.PassID]struct{}{}, nil
	}

	held, err := c.repo.HeldPassIDs(ctx, id, time.Now())
	if err != nil {
		return accountID, nil, fault.Wrap(err, fctx.With(ctx))
	}

	return accountID, held, nil
}

type DriveDecision struct {
	bypass bool
	locks  map[xid.ID][]access.PassRef
	held   map[access.PassID]struct{}
}

func (c *Checker) ResolveDrive(ctx context.Context) (*DriveDecision, error) {
	locks, err := c.repo.DriveFolderLocks(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if len(locks) == 0 || bypasses(ctx) {
		return &DriveDecision{bypass: true}, nil
	}

	_, held, err := c.held(ctx)
	if err != nil {
		return nil, err
	}

	return &DriveDecision{locks: locks, held: held}, nil
}

func (d *DriveDecision) FolderLock(id xid.ID) opt.Optional[access.Lock] {
	if d.bypass {
		return opt.NewEmpty[access.Lock]()
	}

	passes, ok := d.locks[id]
	if !ok {
		return opt.NewEmpty[access.Lock]()
	}

	for _, p := range passes {
		if _, held := d.held[p.ID]; held {
			return opt.NewEmpty[access.Lock]()
		}
	}

	return opt.New(access.Lock{Passes: passes})
}

func (d *DriveDecision) RequireFolder(ctx context.Context, id xid.ID) error {
	if lock, locked := d.FolderLock(id).Get(); locked {
		return lockedError(ctx, lock)
	}
	return nil
}

// CategoryLock reports the passes that would unlock a category the requester
// cannot currently read. Empty means the category is readable.
func (d *Decision) CategoryLock(id category.CategoryID) opt.Optional[access.Lock] {
	if d.bypass {
		return opt.NewEmpty[access.Lock]()
	}

	passes, ok := d.locks[id]
	if !ok {
		return opt.NewEmpty[access.Lock]()
	}

	for _, p := range passes {
		if _, held := d.held[p.ID]; held {
			return opt.NewEmpty[access.Lock]()
		}
	}

	return opt.New(access.Lock{Passes: passes})
}

func (d *Decision) CategoryIDLock(id xid.ID) opt.Optional[access.Lock] {
	if id.IsNil() {
		return opt.NewEmpty[access.Lock]()
	}
	return d.CategoryLock(category.CategoryID(id))
}

func (d *Decision) ThreadLock(t *thread.Thread) opt.Optional[access.Lock] {
	if d.bypass {
		return opt.NewEmpty[access.Lock]()
	}

	if accountID, ok := d.account.Get(); ok && t.Author.ID == accountID {
		return opt.NewEmpty[access.Lock]()
	}

	cat, ok := t.Category.Get()
	if !ok {
		return opt.NewEmpty[access.Lock]()
	}

	return d.CategoryLock(cat.ID)
}

// Redact strips everything but the preview from a thread the requester may
// not read and reports whether it did so.
func (d *Decision) Redact(t *thread.Thread) bool {
	lock, locked := d.ThreadLock(t).Get()
	if !locked {
		return false
	}

	redactThread(t, lock)

	return true
}

func redactThread(t *thread.Thread, lock access.Lock) {
	t.Access = opt.New(lock)
	t.Content = datagraph.Content{}
	t.Short = ""
	t.Assets = nil
	t.WebLink = opt.NewEmpty[link_ref.LinkRef]()
	t.Related = nil
	t.Replies = pagination.Result[*reply.Reply]{
		Size:        t.Replies.Size,
		Results:     0,
		TotalPages:  0,
		CurrentPage: t.Replies.CurrentPage,
		Items:       []*reply.Reply{},
	}
}

func (d *Decision) RedactAll(ts []*thread.Thread) {
	if d.bypass {
		return
	}
	for _, t := range ts {
		d.Redact(t)
	}
}

// RequireThread fails when the requester may not interact with the thread,
// for writes such as replying or reacting where a preview makes no sense.
func (d *Decision) RequireThread(ctx context.Context, t *thread.Thread) error {
	if lock, locked := d.ThreadLock(t).Get(); locked {
		return lockedError(ctx, lock)
	}
	return nil
}

func (d *Decision) RequireCategory(ctx context.Context, id xid.ID) error {
	if lock, locked := d.CategoryIDLock(id).Get(); locked {
		return lockedError(ctx, lock)
	}
	return nil
}

func (d *Decision) ownershipLock(o access_repo.PostOwnership) opt.Optional[access.Lock] {
	if d.bypass {
		return opt.NewEmpty[access.Lock]()
	}

	if accountID, ok := d.account.Get(); ok && o.ThreadAuthor == accountID {
		return opt.NewEmpty[access.Lock]()
	}

	return d.CategoryIDLock(o.CategoryID)
}

// RequirePost fails when the post, thread or reply, sits in a thread the
// requester may not read.
func (c *Checker) RequirePost(ctx context.Context, postID xid.ID) error {
	d, err := c.Resolve(ctx)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	if d.bypass {
		return nil
	}

	owners, err := c.repo.PostThreads(ctx, postID)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	o, ok := owners[postID]
	if !ok {
		return nil
	}

	if lock, locked := d.ownershipLock(o).Get(); locked {
		return lockedError(ctx, lock)
	}

	return nil
}

// AssetReadable reports whether the requester may download an asset. An
// asset is withheld only when every post it is attached to is locked, so a
// file reused in an open thread stays available there.
func (c *Checker) AssetReadable(ctx context.Context, assetID xid.ID) (bool, error) {
	d, err := c.Resolve(ctx)
	if err != nil {
		return false, fault.Wrap(err, fctx.With(ctx))
	}

	if d.bypass {
		return true, nil
	}

	postIDs, err := c.repo.AssetPosts(ctx, assetID)
	if err != nil {
		return false, fault.Wrap(err, fctx.With(ctx))
	}

	if len(postIDs) == 0 {
		return true, nil
	}

	owners, err := c.repo.PostThreads(ctx, postIDs...)
	if err != nil {
		return false, fault.Wrap(err, fctx.With(ctx))
	}

	for _, o := range owners {
		if !d.ownershipLock(o).Ok() {
			return true, nil
		}
	}

	return false, nil
}

func lockedError(ctx context.Context, lock access.Lock) error {
	name := "an access pass"
	if len(lock.Passes) > 0 {
		name = lock.Passes[0].Name
	}

	return fault.Wrap(ErrLocked,
		fctx.With(ctx),
		fmsg.WithDesc("locked", "This content requires "+name+"."),
	)
}

func isPostKind(k datagraph.Kind) bool {
	return k == datagraph.KindThread || k == datagraph.KindReply || k == datagraph.KindPost
}

// FilterItems applies the access rules to mixed search results. Locked
// threads are kept as a title-only preview, locked replies and posts are
// dropped because a reply without its text tells the reader nothing.
func (c *Checker) FilterItems(ctx context.Context, items []datagraph.Item) ([]datagraph.Item, error) {
	d, err := c.Resolve(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if d.bypass {
		return items, nil
	}

	ids := []xid.ID{}
	for _, it := range items {
		if isPostKind(it.GetKind()) {
			ids = append(ids, it.GetID())
		}
	}

	if len(ids) == 0 {
		return items, nil
	}

	owners, err := c.repo.PostThreads(ctx, ids...)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	out := make([]datagraph.Item, 0, len(items))
	for _, it := range items {
		if !isPostKind(it.GetKind()) {
			out = append(out, it)
			continue
		}

		o, ok := owners[it.GetID()]
		if !ok {
			out = append(out, it)
			continue
		}

		lock, locked := d.ownershipLock(o).Get()
		if !locked {
			out = append(out, it)
			continue
		}

		if t, isThread := it.(*thread.Thread); isThread {
			redactThread(t, lock)
			out = append(out, t)
		}
	}

	return out, nil
}

func (c *Checker) FilterMatches(ctx context.Context, matches datagraph.MatchList) (datagraph.MatchList, error) {
	d, err := c.Resolve(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if d.bypass {
		return matches, nil
	}

	ids := []xid.ID{}
	for _, m := range matches {
		if isPostKind(m.Kind) {
			ids = append(ids, m.ID)
		}
	}

	if len(ids) == 0 {
		return matches, nil
	}

	owners, err := c.repo.PostThreads(ctx, ids...)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	out := make(datagraph.MatchList, 0, len(matches))
	for _, m := range matches {
		o, ok := owners[m.ID]
		if !isPostKind(m.Kind) || !ok || !d.ownershipLock(o).Ok() {
			out = append(out, m)
			continue
		}

		if m.Kind == datagraph.KindThread {
			m.Description = ""
			out = append(out, m)
		}
	}

	return out, nil
}

// Covered reports whether a post sits under any pass at all, regardless of
// who is asking. Responses for such posts differ per requester and over time
// as grants change or expire, so HTTP revalidation must not short-circuit.
func (c *Checker) Covered(ctx context.Context, postID xid.ID) (bool, error) {
	locks, err := c.repo.CategoryLocks(ctx)
	if err != nil {
		return false, fault.Wrap(err, fctx.With(ctx))
	}

	if len(locks) == 0 {
		return false, nil
	}

	owners, err := c.repo.PostThreads(ctx, postID)
	if err != nil {
		return false, fault.Wrap(err, fctx.With(ctx))
	}

	o, ok := owners[postID]
	if !ok || o.CategoryID.IsNil() {
		return false, nil
	}

	_, covered := locks[category.CategoryID(o.CategoryID)]
	return covered, nil
}

// FilterWrapped applies FilterItems to any list whose elements carry a
// datagraph item, such as collection entries or profile likes.
func FilterWrapped[T any](ctx context.Context, c *Checker, in []T, item func(T) datagraph.Item) ([]T, error) {
	items := make([]datagraph.Item, len(in))
	for i, v := range in {
		items[i] = item(v)
	}

	kept, err := c.FilterItems(ctx, items)
	if err != nil {
		return nil, err
	}

	if len(kept) == len(in) {
		return in, nil
	}

	keep := make(map[xid.ID]struct{}, len(kept))
	for _, it := range kept {
		keep[it.GetID()] = struct{}{}
	}

	out := make([]T, 0, len(kept))
	for _, v := range in {
		if _, ok := keep[item(v).GetID()]; ok {
			out = append(out, v)
		}
	}

	return out, nil
}
