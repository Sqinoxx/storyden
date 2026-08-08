// Package drive_credentials resolves which Google service account key the
// Drive browser should use: one uploaded via the admin interface, held in the
// settings database, or the static GOOGLE_DRIVE_SERVICE_ACCOUNT_JSON/_FILE
// environment configuration as a fallback for installations without an admin
// willing to hold the key in the database.
package drive_credentials

import (
	"context"
	"io"
	"sync"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/Southclaws/opt"

	"github.com/Southclaws/storyden/app/resources/rbac"
	"github.com/Southclaws/storyden/app/resources/settings"
	"github.com/Southclaws/storyden/app/services/authentication/session"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/infrastructure/gdrive"
)

type Source string

const (
	SourceNone        Source = "none"
	SourceEnvironment Source = "environment"
	SourceUpload      Source = "upload"
)

type Status struct {
	Configured          bool
	Source              Source
	ServiceAccountEmail string
}

// Resolver is the gdrive.Client used across the application. It prefers a
// service account key uploaded via the admin interface and falls back to the
// static, environment-configured client when none has been uploaded.
type Resolver struct {
	settingsRepo *settings.SettingsRepository
	static       gdrive.Client

	mu          sync.RWMutex
	uploadedRaw string
	uploaded    gdrive.Client
}

var _ gdrive.Client = (*Resolver)(nil)

func New(ctx context.Context, cfg config.Config, settingsRepo *settings.SettingsRepository) (*Resolver, error) {
	static, err := gdrive.NewFromConfig(ctx, cfg)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return &Resolver{settingsRepo: settingsRepo, static: static}, nil
}

func (r *Resolver) Enabled() bool {
	return r.active(context.Background()).Enabled()
}

func (r *Resolver) List(ctx context.Context, folderID string, pageToken string) (*gdrive.Listing, error) {
	return r.active(ctx).List(ctx, folderID, pageToken)
}

func (r *Resolver) Get(ctx context.Context, fileID string) (*gdrive.File, error) {
	return r.active(ctx).Get(ctx, fileID)
}

func (r *Resolver) Open(ctx context.Context, f *gdrive.File) (io.ReadCloser, string, error) {
	return r.active(ctx).Open(ctx, f)
}

// active returns the client that should serve this call: the uploaded key if
// one is set, otherwise the static, environment-configured client.
func (r *Resolver) active(ctx context.Context) gdrive.Client {
	raw := r.uploadedJSON(ctx)
	if raw == "" {
		return r.static
	}

	r.mu.RLock()
	if raw == r.uploadedRaw && r.uploaded != nil {
		client := r.uploaded
		r.mu.RUnlock()
		return client
	}
	r.mu.RUnlock()

	client, err := gdrive.NewClient(ctx, []byte(raw))
	if err != nil {
		// The key is validated before it's stored, so this should not happen
		// in practice; fall back rather than break every Drive request.
		return r.static
	}

	r.mu.Lock()
	r.uploadedRaw = raw
	r.uploaded = client
	r.mu.Unlock()

	return client
}

func (r *Resolver) uploadedJSON(ctx context.Context) string {
	set, err := r.settingsRepo.Get(ctx)
	if err != nil {
		return ""
	}

	services, ok := set.Services.Get()
	if !ok {
		return ""
	}

	drive, ok := services.Drive.Get()
	if !ok {
		return ""
	}

	return drive.ServiceAccountJSON.Or("")
}

// Status reports where the active credentials came from, for the admin
// interface.
func (r *Resolver) Status(ctx context.Context) (Status, error) {
	if raw := r.uploadedJSON(ctx); raw != "" {
		email, _ := gdrive.ServiceAccountEmail(ctx, []byte(raw))

		return Status{Configured: true, Source: SourceUpload, ServiceAccountEmail: email}, nil
	}

	if r.static.Enabled() {
		return Status{Configured: true, Source: SourceEnvironment}, nil
	}

	return Status{Configured: false, Source: SourceNone}, nil
}

// Upload validates a service account key and, if it works, stores it and
// makes it the active credentials immediately.
func (r *Resolver) Upload(ctx context.Context, credentials []byte) (Status, error) {
	if err := authorise(ctx); err != nil {
		return Status{}, err
	}

	email, err := gdrive.ServiceAccountEmail(ctx, credentials)
	if err != nil {
		return Status{}, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.InvalidArgument))
	}

	client, err := gdrive.NewClient(ctx, credentials)
	if err != nil {
		return Status{}, fault.Wrap(err,
			fctx.With(ctx),
			ftag.With(ftag.InvalidArgument),
			fmsg.WithDesc("invalid service account key",
				"This key could not be used to connect to the Google Drive API."))
	}

	raw := string(credentials)

	if _, err := r.settingsRepo.Set(ctx, settings.Settings{
		Services: opt.New(settings.ServiceSettings{
			Drive: opt.New(settings.DriveServiceSettings{
				ServiceAccountJSON: opt.New(raw),
			}),
		}),
	}); err != nil {
		return Status{}, fault.Wrap(err, fctx.With(ctx))
	}

	r.mu.Lock()
	r.uploadedRaw = raw
	r.uploaded = client
	r.mu.Unlock()

	return Status{Configured: true, Source: SourceUpload, ServiceAccountEmail: email}, nil
}

// Remove withdraws the uploaded key. The installation falls back to its
// environment configuration, if any.
func (r *Resolver) Remove(ctx context.Context) (Status, error) {
	if err := authorise(ctx); err != nil {
		return Status{}, err
	}

	if _, err := r.settingsRepo.Set(ctx, settings.Settings{
		Services: opt.New(settings.ServiceSettings{
			Drive: opt.New(settings.DriveServiceSettings{
				ServiceAccountJSON: opt.New(""),
			}),
		}),
	}); err != nil {
		return Status{}, fault.Wrap(err, fctx.With(ctx))
	}

	r.mu.Lock()
	r.uploadedRaw = ""
	r.uploaded = nil
	r.mu.Unlock()

	return r.Status(ctx)
}

func authorise(ctx context.Context) error {
	if err := session.Authorise(ctx, nil, rbac.PermissionManageSettings, rbac.PermissionAdministrator); err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	return nil
}
