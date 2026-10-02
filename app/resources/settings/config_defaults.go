package settings

import (
	"github.com/Southclaws/fault"
	"github.com/Southclaws/opt"

	"github.com/Southclaws/storyden/internal/ent"
)

// hydrateConfigDefaults takes an *ent.Setting, maps it to *Settings, and
// injects config.Config defaults for any values not set in the database.
func (d *SettingsRepository) hydrateConfigDefaults(in *ent.Setting) (*Settings, error) {
	settings, err := mapSettings(in)
	if err != nil {
		return nil, fault.Wrap(err)
	}

	d.decryptSecrets(settings)

	d.hydrateClientIPDefaults(settings)
	d.hydrateRateLimitDefaults(settings)
	d.hydrateAssetDefaults(settings)

	return settings, nil
}

func (d *SettingsRepository) hydrateClientIPDefaults(settings *Settings) {
	services, ok := settings.Services.Get()
	if !ok {
		services = ServiceSettings{}
	}

	clientIP, ok := services.ClientIP.Get()
	if !ok {
		services.ClientIP = opt.New(ClientIPServiceSettings{
			ClientIPMode:   opt.New(ClientIPModeRemoteAddr),
			ClientIPHeader: opt.New("X-Real-IP"),
		})
		settings.Services = opt.New(services)
		return
	}

	if !clientIP.ClientIPMode.Ok() {
		clientIP.ClientIPMode = opt.New(ClientIPModeRemoteAddr)
	}
	if !clientIP.ClientIPHeader.Ok() {
		clientIP.ClientIPHeader = opt.New("X-Real-IP")
	}

	services.ClientIP = opt.New(clientIP)
	settings.Services = opt.New(services)
}

func (d *SettingsRepository) hydrateRateLimitDefaults(settings *Settings) {
	services := settings.Services.OrZero()
	rateLimit := services.RateLimit.OrZero()

	if !rateLimit.RateLimit.Ok() {
		rateLimit.RateLimit = opt.New(d.config.RateLimit)
	}

	if !rateLimit.RateLimitPeriod.Ok() {
		rateLimit.RateLimitPeriod = opt.New(d.config.RateLimitPeriod)
	}

	if !rateLimit.RateLimitBucket.Ok() {
		rateLimit.RateLimitBucket = opt.New(d.config.RateLimitBucket)
	}

	if !rateLimit.RateLimitGuestCost.Ok() {
		rateLimit.RateLimitGuestCost = opt.New(d.config.RateLimitGuestCost)
	}

	if !rateLimit.LoginMaxAttempts.Ok() {
		rateLimit.LoginMaxAttempts = opt.New(d.config.LoginMaxAttempts)
	}

	if !rateLimit.LoginLockoutDuration.Ok() {
		rateLimit.LoginLockoutDuration = opt.New(d.config.LoginLockoutDuration)
	}

	services.RateLimit = opt.New(rateLimit)
	settings.Services = opt.New(services)
}

func (d *SettingsRepository) hydrateAssetDefaults(settings *Settings) {
	services, ok := settings.Services.Get()
	if !ok {
		settings.Services = opt.New(ServiceSettings{
			Assets: opt.New(AssetServiceSettings{
				MaxUploadSizeMB:  opt.New(d.config.MaxUploadSizeMB),
				OCRMaxFileSizeMB: opt.New(d.config.OCRMaxFileSizeMB),
				OCRTimeout:       opt.New(d.config.OCRTimeout),
			}),
		})
		return
	}

	assets, ok := services.Assets.Get()
	if !ok {
		services.Assets = opt.New(AssetServiceSettings{
			MaxUploadSizeMB:  opt.New(d.config.MaxUploadSizeMB),
			OCRMaxFileSizeMB: opt.New(d.config.OCRMaxFileSizeMB),
			OCRTimeout:       opt.New(d.config.OCRTimeout),
		})
		settings.Services = opt.New(services)
		return
	}

	if !assets.MaxUploadSizeMB.Ok() {
		assets.MaxUploadSizeMB = opt.New(d.config.MaxUploadSizeMB)
	}
	if !assets.OCRMaxFileSizeMB.Ok() {
		assets.OCRMaxFileSizeMB = opt.New(d.config.OCRMaxFileSizeMB)
	}
	if !assets.OCRTimeout.Ok() {
		assets.OCRTimeout = opt.New(d.config.OCRTimeout)
	}

	services.Assets = opt.New(assets)
	settings.Services = opt.New(services)
}
