package flightlesssomething

import (
	"errors"
	"flag"

	"github.com/csmith/envflag"
)

type Config struct {
	Bind          string
	DataDir       string
	SessionSecret string

	DiscordClientID     string
	DiscordClientSecret string
	DiscordRedirectURL  string

	OpenAIURL    string
	OpenAIApiKey string
	OpenAIModel  string

	AdminUsername string
	AdminPassword string

	Version bool
}

func NewConfig() (*Config, error) {
	config := &Config{}

	flag.StringVar(&config.Bind, "bind", "0.0.0.0:8080", "Bind address and port")
	flag.StringVar(&config.DataDir, "data-dir", "/data", "Path where data would be stored")
	flag.StringVar(&config.SessionSecret, "session-secret", "", "Session secret")

	flag.StringVar(&config.DiscordClientID, "discord-client-id", "", "Discord OAuth2 client ID (see https://discord.com/developers/applications)")
	flag.StringVar(&config.DiscordClientSecret, "discord-client-secret", "", "Discord OAuth2 client secret (see https://discord.com/developers/applications)")
	flag.StringVar(&config.DiscordRedirectURL, "discord-redirect-url", "", "Discord OAuth2 redirect URL (<scheme>://<domain>/login/callback)")

	flag.StringVar(&config.OpenAIURL, "openai-url", "https://api.openai.com/v1", "OpenAI API URL")
	flag.StringVar(&config.OpenAIModel, "openai-model", "gpt-4o", "OpenAI model ID")
	flag.StringVar(&config.OpenAIApiKey, "openai-api-key", "", "OpenAI API Key (leave empty to disable OpenAI integration)")

	flag.StringVar(&config.AdminUsername, "admin-username", "", "Admin username for testing login (optional)")
	flag.StringVar(&config.AdminPassword, "admin-password", "", "Admin password for testing login (optional)")

	flag.BoolVar(&config.Version, "version", false, "prints version of the application")

	envflag.Parse(envflag.WithPrefix("FS_"))

	if config.Version {
		return config, nil
	}

	if config.DataDir == "" {
		return nil, errors.New("missing data-dir argument")
	}
	if config.DiscordClientID == "" {
		return nil, errors.New("missing discord-client-id argument")
	}
	if config.DiscordClientSecret == "" {
		return nil, errors.New("missing discord-client-secret argument")
	}
	if config.DiscordRedirectURL == "" {
		return nil, errors.New("missing discord-redirect-url argument")
	}
	if config.SessionSecret == "" {
		return nil, errors.New("missing session-secret argument")
	}
	if config.OpenAIApiKey != "" {
		if config.OpenAIModel == "" {
			return nil, errors.New("missing openai-model argument")
		}
		if config.OpenAIURL == "" {
			return nil, errors.New("missing openai-url argument")
		}
	}

	return config, nil
}
