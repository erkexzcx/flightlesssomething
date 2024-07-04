package flightlesssomething

import (
	"errors"
	"flag"

	"github.com/csmith/envflag"
)

type Config struct {
	Bind    string
	DataDir string

	DiscordClientID     string
	DiscordClientSecret string
	DiscordRedirectURL  string

	Version bool
}

func NewConfig() (*Config, error) {
	// Define the flags
	bind := flag.String("bind", "0.0.0.0:8080", "Bind address and port")
	dataDir := flag.String("data-dir", "/data", "Path where data would be stored")
	discordClientID := flag.String("discord-client-id", "", "Discord OAuth2 client ID (see https://discord.com/developers/applications)")
	discordClientSecret := flag.String("discord-client-secret", "", "Discord OAuth2 client secret (see https://discord.com/developers/applications)")
	discordRedirectURL := flag.String("discord-redirect-url", "", "Discord OAuth2 redirect URL (<scheme>://<domain>/login/callback)")
	flagVersion := flag.Bool("version", false, "prints version of the application")

	envflag.Parse(envflag.WithPrefix("FS_"))

	// Assign the parsed flag values to the Config struct
	config := &Config{
		Bind:                *bind,
		DataDir:             *dataDir,
		DiscordClientID:     *discordClientID,
		DiscordClientSecret: *discordClientSecret,
		DiscordRedirectURL:  *discordRedirectURL,

		Version: *flagVersion,
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

	return config, nil
}
