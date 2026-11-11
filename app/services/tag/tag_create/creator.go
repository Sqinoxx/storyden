package tag_create

import (
	"context"
	"strings"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"

	"github.com/Southclaws/storyden/app/resources/rbac"
	"github.com/Southclaws/storyden/app/resources/tag/tag_ref"
	"github.com/Southclaws/storyden/app/resources/tag/tag_writer"
	"github.com/Southclaws/storyden/app/services/authentication/session"
)

type Creator struct {
	tagWriter *tag_writer.Writer
}

func New(tagWriter *tag_writer.Writer) *Creator {
	return &Creator{tagWriter: tagWriter}
}

func Authorise(ctx context.Context) error {
	return session.Authorise(ctx, nil, rbac.PermissionCreateTags, rbac.PermissionManageTags)
}

func (c *Creator) Add(ctx context.Context, names ...tag_ref.Name) ([]*tag_ref.Tag, error) {
	if len(names) == 0 {
		return nil, nil
	}

	missing, err := c.tagWriter.Missing(ctx, names...)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if len(missing) > 0 {
		if err := Authorise(ctx); err != nil {
			return nil, fault.Wrap(err,
				fctx.With(ctx),
				fmsg.WithDesc("cannot create tags", "You do not have permission to create new tags: "+strings.Join(missing.Strings(), ", ")),
			)
		}
	}

	tags, err := c.tagWriter.Add(ctx, names...)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return tags, nil
}
