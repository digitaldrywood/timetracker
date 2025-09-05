package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/digitaldrywood/timetracker/internal/github"
	"github.com/digitaldrywood/timetracker/internal/google"
	"github.com/digitaldrywood/timetracker/internal/tracker"
)

const (
	credentialsPath = ".local/credentials.json"
	spreadsheetID   = "14RvNPAyigKffw_Xn0blnOvE4hPNZK6vHkHa0k2wWVaA"
)

func main() {
	var (
		summary = flag.Bool("summary", false, "Show daily summary")
		week    = flag.Bool("week", false, "Show weekly summary")
		add     = flag.Bool("add", false, "Add time entry interactively")
		suggest = flag.Bool("suggest", false, "Generate suggested entries from GitHub activity")
	)
	flag.Parse()

	auth, err := google.NewAuth(credentialsPath)
	if err != nil {
		log.Fatalf("Failed to create auth client: %v", err)
	}

	service, err := auth.GetSheetsService()
	if err != nil {
		log.Fatalf("Failed to get Sheets service: %v", err)
	}

	sheets := google.NewSheetsClient(service, spreadsheetID)

	gh, err := github.NewClient()
	if err != nil {
		log.Fatalf("Failed to create GitHub client: %v", err)
	}

	t := tracker.NewTracker(sheets, gh)

	switch {
	case *summary:
		showDailySummary(t)
	case *week:
		showWeeklySummary(t)
	case *add:
		addTimeEntry(t)
	case *suggest:
		suggestEntries(t)
	default:
		showDailySummary(t)
	}
}

func showDailySummary(t *tracker.Tracker) {
	summary, err := t.GetDailySummary()
	if err != nil {
		log.Fatalf("Failed to get daily summary: %v", err)
	}

	fmt.Println(t.FormatDailySummary(summary))
}

func showWeeklySummary(t *tracker.Tracker) {
	summary, err := t.GetWeekSummary()
	if err != nil {
		log.Fatalf("Failed to get weekly summary: %v", err)
	}

	fmt.Println("=== Weekly Summary ===")
	total := 0.0
	for project, hours := range summary {
		fmt.Printf("%-40s: %.1f hours\n", project, hours)
		total += hours
	}
	fmt.Printf("\nTotal: %.1f hours\n", total)
}

func addTimeEntry(t *tracker.Tracker) {
	reader := bufio.NewReader(os.Stdin)

	entry := google.TimeEntry{}

	fmt.Print("Project: ")
	entry.Project, _ = reader.ReadString('\n')
	entry.Project = strings.TrimSpace(entry.Project)

	fmt.Print("Task: ")
	entry.Task, _ = reader.ReadString('\n')
	entry.Task = strings.TrimSpace(entry.Task)

	fmt.Print("Hours: ")
	hoursStr, _ := reader.ReadString('\n')
	hoursStr = strings.TrimSpace(hoursStr)
	entry.Hours, _ = strconv.ParseFloat(hoursStr, 64)

	fmt.Print("Description: ")
	entry.Description, _ = reader.ReadString('\n')
	entry.Description = strings.TrimSpace(entry.Description)

	entry.Date = time.Now().Format("2006-01-02")

	if err := t.AddTimeEntry(entry); err != nil {
		log.Fatalf("Failed to add time entry: %v", err)
	}

	fmt.Println("Time entry added successfully!")
}

func suggestEntries(t *tracker.Tracker) {
	summary, err := t.GetDailySummary()
	if err != nil {
		log.Fatalf("Failed to get daily summary: %v", err)
	}

	if len(summary.SuggestedEntries) == 0 {
		fmt.Println("No suggested entries based on today's GitHub activity.")
		return
	}

	fmt.Println(t.FormatDailySummary(summary))

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nWould you like to add any of these entries? (y/n): ")
	response, _ := reader.ReadString('\n')

	if strings.ToLower(strings.TrimSpace(response)) != "y" {
		return
	}

	for i, entry := range summary.SuggestedEntries {
		fmt.Printf("\n--- Entry %d ---\n", i+1)
		fmt.Printf("Project: %s\n", entry.Project)
		fmt.Printf("Task: %s\n", entry.Task)

		fmt.Print("Hours worked (or 'skip'): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "skip" {
			continue
		}

		hours, err := strconv.ParseFloat(input, 64)
		if err != nil {
			fmt.Println("Invalid hours, skipping...")
			continue
		}

		entry.Hours = hours

		fmt.Print("Additional description (optional): ")
		desc, _ := reader.ReadString('\n')
		entry.Description = strings.TrimSpace(desc)

		if err := t.AddTimeEntry(entry); err != nil {
			fmt.Printf("Failed to add entry: %v\n", err)
		} else {
			fmt.Println("Entry added successfully!")
		}
	}
}
