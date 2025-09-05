package main

import (
	"fmt"
	"log"

	"github.com/digitaldrywood/timetracker/internal/google"
)

const (
	credentialsPath = ".local/credentials.json"
	spreadsheetID   = "14RvNPAyigKffw_Xn0blnOvE4hPNZK6vHkHa0k2wWVaA"
)

func main() {
	fmt.Println("=== Time Tracker Authentication ===")
	fmt.Println()

	auth, err := google.NewAuth(credentialsPath)
	if err != nil {
		log.Fatalf("Failed to create auth client: %v", err)
	}

	// This will trigger the OAuth flow if needed
	service, err := auth.GetSheetsService()
	if err != nil {
		log.Fatalf("Failed to authenticate: %v", err)
	}

	// Test the connection by getting spreadsheet metadata
	spreadsheet, err := service.Spreadsheets.Get(spreadsheetID).Do()
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