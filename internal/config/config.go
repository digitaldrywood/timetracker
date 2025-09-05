package config

import (
	"fmt"
	"os"
)

type Config struct {
	SpreadsheetID   string
	CredentialsPath string
	TokenPath       string
	OAuthPort       string
	OAuthRedirectURL string
}

func Load() (*Config, error) {
	cfg := &Config{
		SpreadsheetID:   os.Getenv("TIMETRACKER_SPREADSHEET_ID"),
		CredentialsPath: os.Getenv("TIMETRACKER_CREDENTIALS_PATH"),
		TokenPath:       os.Getenv("TIMETRACKER_TOKEN_PATH"),
		OAuthPort:       os.Getenv("TIMETRACKER_OAUTH_PORT"),
		OAuthRedirectURL: os.Getenv("TIMETRACKER_OAUTH_REDIRECT_URL"),
	}

	// Set defaults if not provided
	if cfg.CredentialsPath == "" {
		cfg.CredentialsPath = ".local/credentials.json"
	}
	if cfg.TokenPath == "" {
		cfg.TokenPath = ".local/token.json"
	}
	if cfg.OAuthPort == "" {
		cfg.OAuthPort = "8080"
	}
	if cfg.OAuthRedirectURL == "" {
		cfg.OAuthRedirectURL = "http://localhost:8080/callback"
	}

	// Validate required fields
	if cfg.SpreadsheetID == "" {
		return nil, fmt.Errorf("TIMETRACKER_SPREADSHEET_ID environment variable is required. Please set it in .envrc and run 'direnv allow'")
	}

	return cfg, nil
}