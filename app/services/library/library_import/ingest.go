package library_import

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/Southclaws/opt"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/app/resources/library"
	"github.com/Southclaws/storyden/app/resources/library/node_properties"
	"github.com/Southclaws/storyden/app/resources/library/node_querier"
	"github.com/Southclaws/storyden/app/resources/mark"
	"github.com/Southclaws/storyden/app/resources/tag/tag_ref"
	"github.com/Southclaws/storyden/app/resources/tag/tag_writer"
	"github.com/Southclaws/storyden/app/resources/visibility"
	"github.com/Southclaws/storyden/app/services/asset/asset_upload"
	"github.com/Southclaws/storyden/app/services/library/node_mutate"
	"github.com/Southclaws/storyden/app/services/library/node_property_schema"
	"github.com/Southclaws/storyden/internal/ent"
)

type Ingester struct {
	logger   *slog.Logger
	nodes    *node_mutate.Manager
	querier  *node_querier.Querier
	schemas  *node_property_schema.Updater
	uploader *asset_upload.Uploader
	tags     *tag_writer.Writer
}

func NewIngester(
	logger *slog.Logger,
	nodes *node_mutate.Manager,
	querier *node_querier.Querier,
	schemas *node_property_schema.Updater,
	uploader *asset_upload.Uploader,
	tags *tag_writer.Writer,
) *Ingester {
	return &Ingester{
		logger:   logger,
		nodes:    nodes,
		querier:  querier,
		schemas:  schemas,
		uploader: uploader,
		tags:     tags,
	}
}

type IngestOptions struct {
	Root     string
	Owner    account.AccountID
	Ledger   *Ledger
	Progress func(done, total int, path string)
}

type IngestResult struct {
	ContainersCreated int
	ContainersReused  int
	FilesIngested     int
	FilesSkipped      int
	SchemasApplied    int
	PropertiesApplied int
}

// ApplyVocabulary reconciles the tag vocabulary before any content is written:
// renames carry existing post and node links across, then any missing term is
// created so classification later has a closed set to work against.
func (i *Ingester) ApplyVocabulary(ctx context.Context, v *Vocabulary, dryRun bool) ([]string, error) {
	actions := []string{}

	for from, to := range v.Renames {
		actions = append(actions, "rename "+from+" -> "+to)
		if dryRun {
			continue
		}

		if _, err := i.tags.Rename(ctx, tag_ref.NewName(from), tag_ref.NewName(to)); err != nil {
			if ftag.Get(err) == ftag.NotFound {
				i.logger.Info("vocabulary rename skipped, tag not present", slog.String("tag", from))
				continue
			}
			return nil, fault.Wrap(err, fctx.With(ctx), fmsg.With("failed to rename tag "+from))
		}
	}

	names := make([]tag_ref.Name, 0, len(v.Tags()))
	for _, t := range v.Tags() {
		names = append(names, tag_ref.NewName(t))
		actions = append(actions, "ensure "+t)
	}

	if dryRun {
		return actions, nil
	}

	if _, err := i.tags.Add(ctx, names...); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx), fmsg.With("failed to create vocabulary tags"))
	}

	return actions, nil
}

// Apply runs in four passes because the schema machinery demands that order:
// UpdateChildren is a no-op on a node that has no children yet, and a property
// mutation can only address fields by ID once the schema exists. So containers
// come first, then files, then the child schema, then the values.
func (i *Ingester) Apply(ctx context.Context, plan *Plan, opts IngestOptions) (*IngestResult, error) {
	result := &IngestResult{}

	nodeIDs := map[string]library.NodeID{}

	for _, container := range plan.Containers {
		id, created, err := i.ensureContainer(ctx, container, nodeIDs, opts.Owner)
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx), fmsg.With("failed to create container "+container.Slug))
		}

		nodeIDs[container.Slug] = id

		if created {
			result.ContainersCreated++
		} else {
			result.ContainersReused++
		}
	}

	for n, file := range plan.Files {
		if opts.Progress != nil {
			opts.Progress(n+1, len(plan.Files), file.Entry.Path)
		}

		if opts.Ledger != nil && opts.Ledger.Has(file.Entry.SHA256) {
			result.FilesSkipped++
			continue
		}

		if err := i.ingestFile(ctx, file, nodeIDs, opts); err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx), fmsg.With("failed to ingest "+file.Entry.Path))
		}

		result.FilesIngested++
	}

	schemas := map[string]*library.PropertySchema{}
	for _, container := range plan.Containers {
		schema, err := i.applyChildSchema(ctx, container, nodeIDs[container.Slug])
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx), fmsg.With("failed to set child schema on "+container.Slug))
		}
		if schema != nil && len(schema.Fields) > 0 {
			schemas[container.Slug] = schema
			result.SchemasApplied++
		}
	}

	for _, file := range plan.Files {
		applied, err := i.applyProperties(ctx, file, schemas[file.ParentSlug])
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx), fmsg.With("failed to set properties on "+file.Slug))
		}
		if applied {
			result.PropertiesApplied++
		}
	}

	return result, nil
}

func (i *Ingester) ensureContainer(
	ctx context.Context,
	c PlannedContainer,
	nodeIDs map[string]library.NodeID,
	owner account.AccountID,
) (library.NodeID, bool, error) {
	if existing, err := i.querier.Get(ctx, library.NewKey(c.Slug)); err == nil {
		return library.NodeID(existing.Mark.ID()), false, nil
	} else if !isNotFound(err) {
		return library.NodeID{}, false, fault.Wrap(err, fctx.With(ctx))
	}

	partial := node_mutate.Partial{
		Slug:       opt.New(mark.NewSlugFromName(c.Slug)),
		Visibility: opt.New(visibility.VisibilityPublished),
	}

	if len(c.Tags) > 0 {
		partial.Tags = opt.New(tagNames(c.Tags))
	}

	if parent, ok := nodeIDs[c.ParentSlug]; ok {
		partial.Parent = opt.New(library.NewID(xid.ID(parent)))
	}

	node, err := i.nodes.Create(ctx, owner, c.Name, partial)
	if err != nil {
		return library.NodeID{}, false, fault.Wrap(err, fctx.With(ctx))
	}

	return library.NodeID(node.Mark.ID()), true, nil
}

// isNotFound covers both error shapes a lookup can produce: node_querier
// forwards the raw ent error, while the service layer tags its own.
func isNotFound(err error) bool {
	return ent.IsNotFound(err) || ftag.Get(err) == ftag.NotFound
}

func (i *Ingester) applyChildSchema(ctx context.Context, c PlannedContainer, id library.NodeID) (*library.PropertySchema, error) {
	if len(c.ChildSchema) == 0 || c.FileCount == 0 {
		return nil, nil
	}

	mutations := make(node_properties.FieldSchemaMutations, 0, len(c.ChildSchema))
	for n, field := range c.ChildSchema {
		mutations = append(mutations, &node_properties.SchemaFieldMutation{
			Name: field,
			Type: library.PropertyTypeEnumText,
			Sort: strconv.Itoa(n),
		})
	}

	schema, err := i.schemas.UpdateChildren(ctx, library.NewID(xid.ID(id)), mutations)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return schema, nil
}

// applyProperties reads the node back so the mutation carries the current
// values, otherwise a rerun would blank anything the enrichment pass wrote.
func (i *Ingester) applyProperties(ctx context.Context, file PlannedFile, schema *library.PropertySchema) (bool, error) {
	if len(file.Properties) == 0 || schema == nil || len(schema.Fields) == 0 {
		return false, nil
	}

	node, err := i.querier.Get(ctx, library.NewKey(file.Slug))
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, fault.Wrap(err, fctx.With(ctx))
	}

	current := library.Properties{}
	if table, ok := node.Properties.Get(); ok {
		current = table.Properties
	}

	props := PropertyMutations(schema, current, file.Properties)
	if len(props) == 0 {
		return false, nil
	}

	if _, err := i.nodes.Update(ctx, library.NewKey(file.Slug), node_mutate.Partial{Properties: opt.New(props)}); err != nil {
		return false, fault.Wrap(err, fctx.With(ctx))
	}

	return true, nil
}

func (i *Ingester) ingestFile(ctx context.Context, file PlannedFile, nodeIDs map[string]library.NodeID, opts IngestOptions) error {
	parent, ok := nodeIDs[file.ParentSlug]
	if !ok {
		return fault.New("planned parent container missing", fctx.With(ctx), fmsg.With("parent "+file.ParentSlug+" was never created"))
	}

	assetID, err := i.upload(ctx, file, opts.Root)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	vis, err := visibility.NewVisibility(file.Visibility)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx), fmsg.With("invalid visibility "+file.Visibility))
	}

	partial := node_mutate.Partial{
		Slug:       opt.New(mark.NewSlugFromName(file.Slug)),
		Parent:     opt.New(library.NewID(xid.ID(parent))),
		Visibility: opt.New(vis),
		AssetsAdd:  opt.New([]asset.AssetID{assetID}),
	}

	if len(file.Tags) > 0 {
		partial.Tags = opt.New(tagNames(file.Tags))
	}

	node, err := i.nodes.Create(ctx, opts.Owner, file.Name, partial)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	if opts.Ledger != nil {
		return opts.Ledger.Record(LedgerRecord{
			SHA256:  file.Entry.SHA256,
			Path:    file.Entry.Path,
			NodeID:  node.Mark.ID().String(),
			AssetID: assetID.String(),
			Slug:    node.GetSlug(),
		})
	}

	return nil
}

// upload streams the file straight through the uploader rather than the HTTP
// API, which is what keeps the source tree's several-hundred-megabyte scans
// from hitting MAX_UPLOAD_SIZE_MB.
func (i *Ingester) upload(ctx context.Context, file PlannedFile, root string) (asset.AssetID, error) {
	abs := filepath.Join(root, filepath.FromSlash(file.Entry.Path))

	f, err := os.Open(abs)
	if err != nil {
		return asset.AssetID{}, fault.Wrap(err, fctx.With(ctx))
	}
	defer f.Close()

	a, err := i.uploader.Upload(ctx, f, file.Entry.Size, asset.NewFilename(filepath.Base(file.Entry.Path)), asset_upload.Options{})
	if err != nil {
		return asset.AssetID{}, fault.Wrap(err, fctx.With(ctx))
	}

	return a.ID, nil
}

func tagNames(in []string) tag_ref.Names {
	out := make(tag_ref.Names, 0, len(in))
	for _, t := range in {
		out = append(out, tag_ref.NewName(t))
	}
	return out
}

// PropertyMutations builds a mutation covering every field of the schema.
//
// This has to be exhaustive and carry field IDs: PropertySchema.Split treats
// any field the mutation does not mention as removed, so a name-only mutation
// for one property silently deletes the rest of the schema.
func PropertyMutations(schema *library.PropertySchema, current library.Properties, values map[string]string) library.PropertyMutationList {
	if schema == nil || len(schema.Fields) == 0 {
		return nil
	}

	existing := map[string]string{}
	for _, p := range current {
		existing[p.Field.Name] = p.Value.OrZero()
	}

	out := make(library.PropertyMutationList, 0, len(schema.Fields))
	for _, f := range schema.Fields {
		value, ok := values[f.Name]
		if !ok {
			value = existing[f.Name]
		}

		out = append(out, &library.PropertyMutation{
			ID:    opt.New(f.ID),
			Name:  f.Name,
			Value: value,
			Type:  opt.New(f.Type),
			Sort:  opt.New(f.Sort),
		})
	}

	return out
}

// FixVisibilityOptions configures a SetVisibility run.
type FixVisibilityOptions struct {
	Ledger   *Ledger
	Progress func(done, total int, slug string)
}

type FixVisibilityResult struct {
	Updated int
	Skipped int
}

// SetVisibility retargets every node the ledger recorded to vis. It exists
// because a manifest visibility change (as happened when unlisted turned out
// to exclude nodes from search entirely) should not require re-running the
// whole import — the ledger already knows exactly which nodes this importer
// created, so only those are touched.
func (i *Ingester) SetVisibility(ctx context.Context, vis visibility.Visibility, opts FixVisibilityOptions) (*FixVisibilityResult, error) {
	if opts.Ledger == nil {
		return nil, fault.New("SetVisibility needs the import ledger to know which nodes to touch")
	}

	result := &FixVisibilityResult{}

	records := opts.Ledger.Records()
	for n, rec := range records {
		if opts.Progress != nil {
			opts.Progress(n+1, len(records), rec.Slug)
		}

		node, err := i.querier.Get(ctx, library.NewKey(rec.Slug))
		if err != nil {
			if isNotFound(err) {
				continue
			}
			return nil, fault.Wrap(err, fctx.With(ctx), fmsg.With("failed to load "+rec.Slug))
		}

		if node.Visibility == vis {
			result.Skipped++
			continue
		}

		if _, err := i.nodes.Update(ctx, library.NewKey(rec.Slug), node_mutate.Partial{Visibility: opt.New(vis)}); err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx), fmsg.With("failed to update visibility on "+rec.Slug))
		}

		result.Updated++
	}

	return result, nil
}
