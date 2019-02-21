package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"bitbucket.org/okteto/okteto/backend/logger"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	defaultHostedZone = "okteto.net."
)

func enableConfigFromEnviromentVars() {
	viper.SetEnvPrefix("okteto")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
}

// LoadConfig loads the configuration from the different possible locations
func LoadConfig() {
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		logger.Fatal(errors.Wrap(err, "Failed to read the config"))
	}

	enableConfigFromEnviromentVars()
}

// GetDBConnectionString returns a connection string to the DB
func GetDBConnectionString() (string, string) {
	driver := viper.GetString("database.driver")
	dbinfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		viper.GetString("database.host"),
		viper.GetString("database.port"),
		viper.GetString("database.user"),
		viper.GetString("database.password"),
		viper.GetString("database.database"))
	return driver, dbinfo
}

// GetBaseURL returns the base URL of the api service
func GetBaseURL() string {
	return viper.GetString("api.url")
}

// GetAPIURL returns the base URL of the api service
func GetAPIURL() string {
	return fmt.Sprintf("%s/api/v1", viper.GetString("api.url"))
}

// IsDNSConfigured returns if DNS provider is configured
func IsDNSConfigured() bool {
	return viper.GetString("aws.access_key") != ""
}

// GetDNSCredentials returns the AWS credentials used for creating DNS entries
func GetDNSCredentials() (string, string, string) {
	zone := viper.GetString("aws.hostedzone")
	if zone == "" {
		zone = defaultHostedZone
	}

	return zone, viper.GetString("aws.access_key"), viper.GetString("aws.secret_key")
}

// GetSQLTrace returns True if sql tracing is to be enabled
func GetSQLTrace() bool {
	return viper.GetBool("database.trace")
}

// GetAuthSecret returns the shared secret used between api and tests to
// create users
func GetAuthSecret() string {
	return viper.GetString("api.secret")
}

//GetMailgunCredentials returns the mail domain and the API key needed to use mailgun
func GetMailgunCredentials() (string, string) {
	return viper.GetString("mail.domain"), viper.GetString("mail.api")
}

//GetNotificationEmail returns the email to use when sending notifications
func GetNotificationEmail() string {
	notification := viper.GetString("mail.notification")
	if notification == "" {
		notification = "hello@okteto.com"
	}

	return notification
}

// GetGoogleAuthID returns the Client ID used for google-based auth
func GetGoogleAuthID() string {
	return viper.GetString("google.clientid")
}

// GetCertificateSecret returns the secret for certificates
func GetCertificateSecret() string {
	return viper.GetString("certificate.secret")
}

// GetCertificateNamespace returns the namespace for certificates
func GetCertificateNamespace() string {
	return viper.GetString("certificate.namespace")
}

// GetSlackWebhook returns the webhook used to notify of private activity
func GetSlackWebhook() string {
	return viper.GetString("slack.webhook")
}

// AnalyticsEnabled returns true if analytics are enabled for the installation
func AnalyticsEnabled() bool {
	return viper.GetBool("analytics")
}

// GetSentry returns the sentryDSN and the environment
func GetSentry() (string, string) {
	return os.Getenv("SENTRY_DSN"), os.Getenv("SENTRY_ENVIRONMENT")
}

// GetMixpanel returns the mixpanel key
func GetMixpanel() string {
	return viper.GetString("mixpanel")
}

// GetGithubApp returns the ID of the Github application and the private key
func GetGithubApp() (int, []byte, error) {
	app, err := strconv.Atoi(viper.GetString("github.appid"))
	if err != nil {
		return 0, nil, err
	}

	return app, []byte(viper.GetString("github.privatekey")), nil
}

// GetClusterName returns the name of the default cluster
func GetClusterName() string {
	return viper.GetString("cluster.name")
}

// GetInClusterConfiguration returns the endpoint and cacert of the incluster configuration
func GetInClusterConfiguration() (string, string) {
	return viper.GetString("cluster.endpoint"), viper.GetString("cluster.cacert")
}

// GetInClusterIngressConfiguration returns the ingress domain, TLS type, certificate name and certificate namespace
func GetInClusterIngressConfiguration() (string, string, string, string) {
	return viper.GetString("cluster.ingress.domain"),
		viper.GetString("cluster.ingress.tls.type"),
		viper.GetString("cluster.ingress.tls.namespace"),
		viper.GetString("cluster.ingress.tls.secret")
}

// InClusterIngressEnabled returns true if the current cluster is configured to use an ingress
func InClusterIngressEnabled() bool {
	return viper.GetBool("cluster.ingress.enabled")
}

// UseInClusterConfig returns true if we should use the current cluster
func UseInClusterConfig() bool {
	return viper.GetBool("cluster.enabled")
}

// GetEnvironmentName returns the environment name of the deployment.
// This is used for analytics tags
func GetEnvironmentName() string {
	return viper.GetString("environment")
}
