package config

import (
	"os"
	"testing"
)

func TestApi(t *testing.T) {
	LoadConfig()
	if GetAPIURL() == "" {
		t.Errorf("GetAPIURL returned an empty value")
	}

	if GetAuthSecret() == "" {
		t.Errorf("GetAuthSecret returned an empty value")
	}
}

func TestDNSCredentials(t *testing.T) {
	os.Setenv("OKTETO_AWS_ACCESS_KEY", "accessKey123")
	os.Setenv("OKTETO_AWS_SECRET_KEY", "secretKey456")
	LoadConfig()

	zone, accessKey, secretKey := GetDNSCredentials()

	if accessKey != "accessKey123" {
		t.Errorf("the accessKey value wasn't retrieved from the env var")
	}

	if secretKey != "secretKey456" {
		t.Errorf("the accessKey value wasn't retrieved from the env var")
	}

	if zone == "" {
		t.Errorf("the zone value was empty")
	}

}

func TestGoogleCredentials(t *testing.T) {
	LoadConfig()
	os.Setenv("OKTETO_GOOGLE_CLIENTID", "secret-123456")
	LoadConfig()
	aud := GetGoogleAuthID()
	if aud != "secret-123456" {
		t.Errorf("the google auth id value wasn't retrieved from the env var, it was : %s", aud)
	}
}

func TestSlackWebhook(t *testing.T) {
	LoadConfig()
	webhook := GetSlackWebhook()
	if webhook != "" {
		t.Errorf("didn't received an empty webhook by default")
	}

	os.Setenv("OKTETO_SLACK_WEBHOOK", "https://123456")
	LoadConfig()
	webhook = GetSlackWebhook()
	if webhook != "https://123456" {
		t.Errorf("received an empty webhook")
	}
}

func TestGithubCredentials(t *testing.T) {
	LoadConfig()
	os.Setenv("OKTETO_GITHUB_APPID", "123456")
	appid, _, err := GetGithubApp()
	if err != nil {
		t.Fatalf(err.Error())
	}

	if appid != 123456 {
		t.Errorf("the github app id value wasn't retrieved from the env, it was : %d", appid)
	}
}
func TestAnalytics(t *testing.T) {
	LoadConfig()
	if AnalyticsEnabled() {
		t.Errorf("Analytics are not disabled by default")
	}

	os.Setenv("OKTETO_ANALYTICS", "true")
	if !AnalyticsEnabled() {
		t.Errorf("Analytics are not enabled")
	}

	os.Setenv("OKTETO_ANALYTICS", "1")
	if !AnalyticsEnabled() {
		t.Errorf("Analytics are not enabled")
	}

	os.Setenv("OKTETO_ANALYTICS", "")
	if AnalyticsEnabled() {
		t.Errorf("Analytics are not disabled by empty env var")
	}

}
