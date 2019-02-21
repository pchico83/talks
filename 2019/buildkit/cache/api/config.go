package api

import (
	"bitbucket.org/okteto/okteto/backend/config"
	restful "github.com/emicklei/go-restful"
)

// PublicConfig contains the public configuration values, typically used by the frontend
type PublicConfig struct {
	// GoogleSecretID is the AUD used to verify google accounts
	GoogleSecretID string `json:"google"`

	// Analytics true if analytics is enabled
	Analytics bool `json:"analytics"`

	// Sentry returns the sentry configuration
	Sentry *SentryConfig `json:"sentry"`

	// Mixpanel returns the mixpanel token or empty
	Mixpanel string `json:"mixpanel"`

	// Environment returns the environment name
	Environment string `json:"environment"`
}

// SentryConfig contains the DSN and environment for sentry
type SentryConfig struct {
	DSN         string `json:"dsn"`
	Environment string `json:"environment"`
}

var publicConfig *PublicConfig

func (a *API) getConfig(request *restful.Request, response *restful.Response) {
	if publicConfig == nil {
		dsn, environment := config.GetSentry()
		publicConfig = &PublicConfig{
			GoogleSecretID: config.GetGoogleAuthID(),
			Analytics:      config.AnalyticsEnabled(),
			Sentry: &SentryConfig{
				DSN:         dsn,
				Environment: environment,
			},
			Mixpanel:    config.GetMixpanel(),
			Environment: config.GetEnvironmentName(),
		}

		if publicConfig.Sentry.Environment == "" {
			publicConfig.Sentry.Environment = publicConfig.Environment
		}
	}

	response.WriteEntity(publicConfig)
}
