package github

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GetLocalCommits finds commits in local git repositories
func (c *Client) GetLocalCommits(basePaths []string, since time.Time) ([]Commit, error) {
	var allCommits []Commit
	
	// Default paths to check if none provided
	if len(basePaths) == 0 {
		home, _ := os.UserHomeDir()
		basePaths = []string{
			filepath.Join(home, "projects"),
			filepath.Join(home, "code"),
			filepath.Join(home, "dev"),
			filepath.Join(home, "src"),
			filepath.Join(home, "work"),
			filepath.Join(home, "repos"),
		}
	}
	
	// Find all git repositories
	gitRepos := []string{}
	for _, basePath := range basePaths {
		if _, err := os.Stat(basePath); os.IsNotExist(err) {
			continue
		}
		
		err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors
			}
			
			// Check if this is a .git directory
			if info.IsDir() && info.Name() == ".git" {
				repoPath := filepath.Dir(path)
				gitRepos = append(gitRepos, repoPath)
				return filepath.SkipDir // Don't recurse into .git
			}
			
			// Skip common non-project directories
			if info.IsDir() && (info.Name() == "node_modules" || info.Name() == "vendor" || info.Name() == ".cache") {
				return filepath.SkipDir
			}
			
			return nil
		})
		
		if err != nil {
			continue
		}
	}
	
	// Get commits from each repo
	sinceStr := since.Format("2006-01-02")
	for _, repoPath := range gitRepos {
		commits, err := c.getLocalRepoCommits(repoPath, sinceStr)
		if err != nil {
			continue
		}
		allCommits = append(allCommits, commits...)
	}
	
	return allCommits, nil
}

func (c *Client) getLocalRepoCommits(repoPath string, since string) ([]Commit, error) {
	// Get the repository name from the path
	repoName := filepath.Base(repoPath)
	
	// Try to get remote URL to determine full repo name
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err == nil {
		url := strings.TrimSpace(string(output))
		// Parse GitHub URL to get owner/repo
		if strings.Contains(url, "github.com") {
			parts := strings.Split(url, "github.com")
			if len(parts) > 1 {
				repoPath := strings.TrimPrefix(parts[1], ":")
				repoPath = strings.TrimPrefix(repoPath, "/")
				repoPath = strings.TrimSuffix(repoPath, ".git")
				if repoPath != "" {
					repoName = repoPath
				}
			}
		}
	}
	
	// Get commits using git log
	cmd = exec.Command("git", "log", 
		"--all",  // Check all branches
		"--since="+since,
		"--author="+c.username,
		"--pretty=format:%H|%s|%aI",
		"--no-merges")
	cmd.Dir = repoPath
	
	output, err = cmd.Output()
	if err != nil || len(output) == 0 {
		return nil, nil
	}
	
	var commits []Commit
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			continue
		}
		
		authorDate, err := time.Parse(time.RFC3339, parts[2])
		if err != nil {
			continue
		}
		
		commits = append(commits, Commit{
			SHA:        parts[0][:7], // Short SHA
			Message:    parts[1],
			URL:        fmt.Sprintf("file://%s/commit/%s", repoPath, parts[0]),
			Repository: repoName + " (local)",
			AuthorDate: authorDate,
		})
	}
	
	return commits, nil
}

// GetTodayLocalCommits gets local commits from today
func (c *Client) GetTodayLocalCommits() ([]Commit, error) {
	today := time.Now().Truncate(24 * time.Hour)
	return c.GetLocalCommits(nil, today)
}