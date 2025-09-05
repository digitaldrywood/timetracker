package tracker

import (
	"fmt"
	"strings"
	"time"

	"github.com/digitaldrywood/timetracker/internal/github"
	"github.com/digitaldrywood/timetracker/internal/google"
)

type Tracker struct {
	sheets *google.SheetsClient
	github *github.Client
}

type DailySummary struct {
	Date             string
	Commits          []github.Commit
	PullRequests     []github.PullRequest
	ExistingEntries  []google.TimeEntry
	SuggestedEntries []google.TimeEntry
}

func NewTracker(sheets *google.SheetsClient, github *github.Client) *Tracker {
	return &Tracker{
		sheets: sheets,
		github: github,
	}
}

func (t *Tracker) GetDailySummary() (*DailySummary, error) {
	today := time.Now().Format("2006-01-02")

	commits, err := t.github.GetTodayCommits()
	if err != nil {
		return nil, fmt.Errorf("failed to get commits: %v", err)
	}
	
	// Also get local commits
	localCommits, err := t.github.GetTodayLocalCommits()
	if err == nil && len(localCommits) > 0 {
		commits = append(commits, localCommits...)
	}

	prs, err := t.github.GetTodayPullRequests()
	if err != nil {
		return nil, fmt.Errorf("failed to get pull requests: %v", err)
	}

	existingEntries, err := t.sheets.GetTodayEntries()
	if err != nil {
		return nil, fmt.Errorf("failed to get existing entries: %v", err)
	}

	suggestedEntries := t.generateSuggestedEntries(commits, prs)

	return &DailySummary{
		Date:             today,
		Commits:          commits,
		PullRequests:     prs,
		ExistingEntries:  existingEntries,
		SuggestedEntries: suggestedEntries,
	}, nil
}

func (t *Tracker) generateSuggestedEntries(commits []github.Commit, prs []github.PullRequest) []google.TimeEntry {
	projectMap := make(map[string]*google.TimeEntry)
	today := time.Now().Format("2006-01-02")

	for _, commit := range commits {
		project := commit.Repository
		if entry, exists := projectMap[project]; exists {
			entry.GitCommits += fmt.Sprintf("\n- %s", strings.Split(commit.Message, "\n")[0])
		} else {
			projectMap[project] = &google.TimeEntry{
				Date:        today,
				Project:     project,
				Task:        "Development",
				Hours:       0,
				Description: "",
				GitCommits:  fmt.Sprintf("- %s", strings.Split(commit.Message, "\n")[0]),
				GitPRs:      "",
			}
		}
	}

	for _, pr := range prs {
		project := pr.Repository
		if entry, exists := projectMap[project]; exists {
			entry.GitPRs += fmt.Sprintf("\n- PR #%d: %s", pr.Number, pr.Title)
		} else {
			projectMap[project] = &google.TimeEntry{
				Date:        today,
				Project:     project,
				Task:        "Code Review",
				Hours:       0,
				Description: "",
				GitCommits:  "",
				GitPRs:      fmt.Sprintf("- PR #%d: %s", pr.Number, pr.Title),
			}
		}
	}

	var entries []google.TimeEntry
	for _, entry := range projectMap {
		entries = append(entries, *entry)
	}

	return entries
}

func (t *Tracker) AddTimeEntry(entry google.TimeEntry) error {
	return t.sheets.AppendTimeEntry(entry)
}

func (t *Tracker) GetWeekSummary() (map[string]float64, error) {
	entries, err := t.sheets.GetWeekEntries()
	if err != nil {
		return nil, err
	}

	summary := make(map[string]float64)
	for _, entry := range entries {
		summary[entry.Project] += entry.Hours
	}

	return summary, nil
}

func (t *Tracker) FormatDailySummary(summary *DailySummary) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("=== Time Tracking Summary for %s ===\n\n", summary.Date))

	if len(summary.ExistingEntries) > 0 {
		output.WriteString("📊 Existing Entries:\n")
		for _, entry := range summary.ExistingEntries {
			output.WriteString(fmt.Sprintf("  • %s - %s (%.1f hours)\n", entry.Project, entry.Task, entry.Hours))
		}
		output.WriteString("\n")
	}

	if len(summary.Commits) > 0 {
		output.WriteString("💻 Today's Commits:\n")
		commitsByRepo := make(map[string][]github.Commit)
		for _, commit := range summary.Commits {
			commitsByRepo[commit.Repository] = append(commitsByRepo[commit.Repository], commit)
		}

		for repo, commits := range commitsByRepo {
			output.WriteString(fmt.Sprintf("  %s:\n", repo))
			for _, commit := range commits {
				message := strings.Split(commit.Message, "\n")[0]
				if len(message) > 60 {
					message = message[:57] + "..."
				}
				output.WriteString(fmt.Sprintf("    - %s\n", message))
			}
		}
		output.WriteString("\n")
	}

	if len(summary.PullRequests) > 0 {
		output.WriteString("🔄 Pull Requests:\n")
		for _, pr := range summary.PullRequests {
			output.WriteString(fmt.Sprintf("  • %s PR #%d: %s [%s]\n",
				pr.Repository, pr.Number, pr.Title, pr.State))
		}
		output.WriteString("\n")
	}

	if len(summary.SuggestedEntries) > 0 {
		output.WriteString("💡 Suggested Time Entries:\n")
		for i, entry := range summary.SuggestedEntries {
			output.WriteString(fmt.Sprintf("%d. Project: %s\n", i+1, entry.Project))
			output.WriteString(fmt.Sprintf("   Task: %s\n", entry.Task))
			if entry.GitCommits != "" {
				output.WriteString(fmt.Sprintf("   Commits:%s\n", entry.GitCommits))
			}
			if entry.GitPRs != "" {
				output.WriteString(fmt.Sprintf("   PRs:%s\n", entry.GitPRs))
			}
			output.WriteString("\n")
		}
	}

	return output.String()
}
