package main

import (
	"fmt"
	"log"

	"github.com/digitaldrywood/timetracker/internal/config"
	"github.com/digitaldrywood/timetracker/internal/google"
)

func main() {
	fmt.Println("=== Time Tracker Authentication ===")
	fmt.Println()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	auth, err := google.NewAuth(cfg.CredentialsPath, cfg.TokenPath, cfg.OAuthRedirectURL)
	if err != nil {
		log.Fatalf("Failed to create auth client: %v", err)
	}

	// This will trigger the OAuth flow if needed
	service, err := auth.GetSheetsService()
	if err != nil {
		log.Fatalf("Failed to authenticate: %v", err)
	}

	// Test the connection by getting spreadsheet metadata
	spreadsheet, err := service.Spreadsheets.Get(cfg.SpreadsheetID).Do()
	if err != nil {
		log.Fatalf("Failed to access spreadsheet: %v", err)
	}

	fmt.Println("✅ Authentication successful!")
	fmt.Printf("📊 Connected to spreadsheet: %s\n", spreadsheet.Properties.Title)
	fmt.Println()
	fmt.Println("You can now use the timetracker commands:")
	fmt.Println("  make summary  - Show today's summary")
	fmt.Println("  make week     - Show weekly summary")
	fmt.Println("  make add      - Add time entry")
	fmt.Println("  make suggest  - Get suggestions from GitHub")
}