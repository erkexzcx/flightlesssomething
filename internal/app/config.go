package app

import (
	"errors"
	"flag"
	"os"

	"github.com/peterbourgon/ff/v3"
)

// Config holds the application configuration
type Config struct {
	Bind          string
	DataDir       string
	SessionSecret string

	DiscordClientID     string
	DiscordClientSecret string
	DiscordRedirectURL  string

	AdminUsername string
	AdminPassword string

	Version bool
}

// NewConfig parses configuration from command line and environment variables
func NewConfig() (*Config, error) {
	config := &Config{}

	fs := flag.NewFlagSet("flightlesssomething", flag.ExitOnError)
	fs.StringVar(&config.Bind, "bind", "0.0.0.0:5000", "Bind address and port")
	fs.StringVar(&config.DataDir, "data-dir", "/data", "Path where data would be stored")
	fs.StringVar(&config.SessionSecret, "session-secret", "", "Session secret")

	fs.StringVar(&config.DiscordClientID, "discord-client-id", "", "Discord OAuth2 client ID")
	fs.StringVar(&config.DiscordClientSecret, "discord-client-secret", "", "Discord OAuth2 client secret")
	fs.StringVar(&config.DiscordRedirectURL, "discord-redirect-url", "", "Discord OAuth2 redirect URL")

	fs.StringVar(&config.AdminUsername, "admin-username", "", "Admin username for authentication")
	fs.StringVar(&config.AdminPassword, "admin-password", "", "Admin password for authentication")

	fs.BoolVar(&config.Version, "version", false, "Print version and exit")

	if err := ff.Parse(fs, os.Args[1:], ff.WithEnvVarPrefix("FS")); err != nil {
		return nil, err
	}

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
	if config.AdminUsername == "" {
		return nil, errors.New("missing admin-username argument")
	}
	if config.AdminPassword == "" {
		return nil, errors.New("missing admin-password argument")
	}

	return config, nil
}
