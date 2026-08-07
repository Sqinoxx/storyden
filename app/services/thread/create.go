package thread

import (
	"context"

	"github.com/Southclaws/dt"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/Southclaws/opt"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/datagraph"
	"github.com/Southclaws/storyden/app/resources/post/category"
	"github.com/Southclaws/storyden/app/resources/post/thread"
	"github.com/Southclaws/storyden/app/resources/post/thread_writer"
	"github.com/Southclaws/storyden/app/resources/rbac"
	"github.com/Southclaws/storyden/app/resources/tag/tag_ref"
	"github.com/Southclaws/storyden/app/resources/visibility"
	"github.com/Southclaws/storyden/app/services/authentication/session"
	"github.com/Southclaws/storyden/app/services/moderation/checker"
	"github.com/Southclaws/storyden/lib/plugin/rpc"
)

func (s *service) Create(ctx context.Context,
	title string,
	authorID account.AccountID,
	meta map[string]any,
	partial Partial,
) (*thread.Thread, error) {
	if err := authoriseMutation(ctx, partial); err != nil {
		return nil, err
	}

	if err := s.authoriseCategoryForCreate(ctx, partial); err != nil {
		return nil, err
	}

	if content, ok := partial.Content.Get(); ok {
		stable, err := datagraph.NewRichTextWithNewBlocks(content)
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}
		partial.Content = opt.New(stable.Content)
	}

	opts := partial.Opts()
	opts = append(opts,
		thread_writer.WithMeta(meta),
	)
	opts = s.appendDerivedAssetOpts(ctx, opts, partial)

	// Small hack: default to zero-value of content, which is actually not zero
	// it's <body></body>. Why? who knows... oh, me, yes I should know. I don't.
	if !partial.Content.Ok() {
		c, _ := datagraph.NewRichText("")
		opts = append(opts, thread_writer.WithContent(c))
	}

	if u, ok := partial.URL.Get(); ok {
		ln, err := s.fetcher.Fetch(ctx, u)
		if err == nil {
			opts = append(opts, thread_writer.WithLink(xid.ID(ln.ID)))
		}
	}

	if tags, ok := partial.Tags.Get(); ok {
		newTags, err := s.tagWriter.Add(ctx, tags...)
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}

		tagIDs := dt.Map(newTags, func(t *tag_ref.Tag) tag_ref.ID { return t.ID })

		opts = append(opts, thread_writer.WithTagsAdd(tagIDs...))
	}

	thr, err := s.threadWriter.Create(ctx,
		title,
		authorID,
		opts...,
	)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx), fmsg.With("failed to create thread"))
	}

	if content, ok := partial.Content.Get(); ok && thr.Visibility == visibility.VisibilityPublished {
		result, err := s.cpm.CheckContent(ctx, xid.ID(thr.ID), datagraph.KindThread, title, content)
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}

		if result.Action == checker.ActionReport {
			thr, err = s.threadWriter.Update(ctx, thr.ID, thread_writer.WithVisibility(visibility.VisibilityReview))
			if err != nil {
				return nil, fault.Wrap(err, fctx.With(ctx))
			}
		}
	}

	if err := s.cache.Invalidate(ctx, xid.ID(thr.ID)); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if thr.Visibility == visibility.VisibilityPublished {
		s.bus.Publish(ctx, &rpc.EventThreadPublished{
			ID: thr.ID,
		})
	}

	// TODO: Do this using event consumer.
	s.mentioner.Send(ctx, authorID, *datagraph.NewRef(thr), thr.Content.References()...)

	return thr, nil
}

func authoriseMutation(ctx context.Context, partial Partial) error {
	if partial.Pinned.Ok() {
		err := session.Authorise(ctx, nil, rbac.PermissionManagePosts)
		if err != nil {
			return fault.Wrap(err,
				fctx.With(ctx),
				fmsg.WithDesc("pinned state", "You do not have permission to create a pinned thread."),
			)
		}
	}

	return nil
}

// authoriseCategoryForCreate enforces the leaf-category rule for normal users:
//   - Admins / users with ManageCategories permission may post in any category
//     (including parent categories) or without a category.
//   - All other authenticated users MUST supply a category, and that category
//     must be a leaf (i.e. it has no child categories).
func (s *service) authoriseCategoryForCreate(ctx context.Context, partial Partial) error {
	// If the caller has ManageCategories (admin / mod), skip all restrictions.
	if err := session.Authorise(ctx, nil, rbac.PermissionManageCategories); err == nil {
		return nil
	}

	// Normal users must provide a category.
	catID, ok := partial.Category.Get()
	if !ok {
		return fault.New(
			"a category is required",
			fctx.With(ctx),
			ftag.With(ftag.InvalidArgument),
			fmsg.WithDesc("category required", "You must select a category before creating a thread."),
		)
	}

	// The chosen category must be a leaf (no sub-categories).
	isLeaf, err := s.categoryRepo.IsLeaf(ctx, category.CategoryID(catID))
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	if !isLeaf {
		return fault.New(
			"category is not a leaf category",
			fctx.With(ctx),
			ftag.With(ftag.InvalidArgument),
			fmsg.WithDesc("not a leaf category", "Threads can only be created in the deepest subcategory. Please select a subcategory that has no further sub-categories."),
		)
	}

	return nil
}
