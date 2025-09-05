package github

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Client struct {
	username string
}

type Commit struct {
	SHA        string
	Message    string
	URL        string
	Repository string
	AuthorDate time.Time
}

type PullRequest struct {
	Number     int
	Title      string
	URL        string
	Repository string
	State      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func NewClient() (*Client, error) {
	username, err := getCurrentUser()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %v", err)
	}

	return &Client{username: username}, nil
}

func getCurrentUser() (string, error) {
	cmd := exec.Command("gh", "api", "user", "--jq", ".login")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current user from gh: %v", err)
	}

	return strings.TrimSpace(string(output)), nil
}

func (c *Client) GetTodayCommits() ([]Commit, error) {
	today := time.Now().Format("2006-01-02")
	return c.GetCommitsSince(today)
}

func (c *Client) GetCommitsSince(since string) ([]Commit, error) {
	repos, err := c.getRecentRepositories()
	if err != nil {
		return nil, err
	}

	var allCommits []Commit

	for _, repo := range repos {
		commits, err := c.getRepositoryCommits(repo, since)
		if err != nil {
			continue
		}
		allCommits = append(allCommits, commits...)
	}

	return allCommits, nil
}

func (c *Client) getRecentRepositories() ([]string, error) {
	// Use GitHub events API to find ALL repos the user has been active in
	cmd := exec.Command("gh", "api", fmt.Sprintf("users/%s/events/public", c.username), "--paginate", "--jq", ".[].repo.name")
	output, err := cmd.Output()
	if err != nil {
		// Fallback to owned repos if events fail
		return c.getOwnedRepositories()
	}

	// Parse unique repo names from events
	repoMap := make(map[string]bool)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		repo := strings.TrimSpace(line)
		if repo != "" {
			repoMap[repo] = true
		}
	}
	
	// Also get owned repos to ensure we don't miss any
	ownedRepos, err := c.getOwnedRepositories()
	if err == nil {
		for _, repo := range ownedRepos {
			repoMap[repo] = true
		}
	}
	
	// Convert map to slice
	var repoNames []string
	for repo := range repoMap {
		repoNames = append(repoNames, repo)
	}

	return repoNames, nil
}

func (c *Client) getOwnedRepositories() ([]string, error) {
	cmd := exec.Command("gh", "repo", "list", "--limit", "100", "--json", "nameWithOwner")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %v", err)
	}

	var repos []struct {
		NameWithOwner string `json:"nameWithOwner"`
	}

	if err := json.Unmarshal(output, &repos); err != nil {
		return nil, fmt.Errorf("failed to parse repository list: %v", err)
	}

	var repoNames []string
	for _, r := range repos {
		repoNames = append(repoNames, r.NameWithOwner)
	}

	return repoNames, nil
}

func (c *Client) getRepositoryCommits(repo string, since string) ([]Commit, error) {
	// Try to get commits using gh repo view to find default branch
	// Then use git log style command which is more reliable
	cmd := exec.Command("gh", "repo", "view", repo, "--json", "defaultBranchRef")
	branchOutput, err := cmd.Output()
	if err != nil {
		return nil, nil
	}
	
	var branchInfo struct {
		DefaultBranchRef struct {
			Name string `json:"name"`
		} `json:"defaultBranchRef"`
	}
	
	if err := json.Unmarshal(branchOutput, &branchInfo); err != nil {
		return nil, nil
	}
	
	branch := branchInfo.DefaultBranchRef.Name
	if branch == "" {
		branch = "main" // fallback
	}
	
	// Use gh api to get commits with the correct branch
	query := fmt.Sprintf(`
		query {
			repository(owner: "%s", name: "%s") {
				ref(qualifiedName: "refs/heads/%s") {
					target {
						... on Commit {
							history(first: 100, since: "%sT00:00:00Z") {
								edges {
									node {
										oid
										message
										url
										authoredDate
										author {
											user {
												login
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	`, strings.Split(repo, "/")[0], strings.Split(repo, "/")[1], branch, since)

	cmd = exec.Command("gh", "api", "graphql", "-f", fmt.Sprintf("query=%s", query))
	output, err := cmd.Output()
	if err != nil {
		return nil, nil
	}

	var result struct {
		Data struct {
			Repository struct {
				Ref struct {
					Target struct {
						History struct {
							Edges []struct {
								Node struct {
									OID          string    `json:"oid"`
									Message      string    `json:"message"`
									URL          string    `json:"url"`
									AuthoredDate time.Time `json:"authoredDate"`
									Author       struct {
										User struct {
											Login string `json:"login"`
										} `json:"user"`
									} `json:"author"`
								} `json:"node"`
							} `json:"edges"`
						} `json:"history"`
					} `json:"target"`
				} `json:"ref"`
			} `json:"repository"`
		} `json:"data"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, nil
	}

	var commits []Commit
	for _, edge := range result.Data.Repository.Ref.Target.History.Edges {
		// Filter by author to only include current user's commits
		if edge.Node.Author.User.Login == c.username {
			commits = append(commits, Commit{
				SHA:        edge.Node.OID,
				Message:    edge.Node.Message,
				URL:        edge.Node.URL,
				Repository: repo,
				AuthorDate: edge.Node.AuthoredDate,
			})
		}
	}

	return commits, nil
}

func (c *Client) GetRecentPullRequests(days int) ([]PullRequest, error) {
	// Get recent repos that might have PRs
	repos, err := c.getRecentRepositories()
	if err != nil {
		return nil, err
	}
	
	var allPRs []PullRequest
	cutoffDate := time.Now().AddDate(0, 0, -days)
	
	// Check each repo for recent PRs
	for _, repo := range repos {
		cmd := exec.Command("gh", "pr", "list",
			"--repo", repo,
			"--author", c.username,
			"--state", "all",
			"--json", "number,title,url,state,createdAt,updatedAt",
			"--limit", "20")
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			// Skip repos with errors (might not have PR access)
			continue
		}
		
		// Handle empty results
		if len(output) == 0 || string(output) == "[]\n" || string(output) == "[]" {
			continue
		}
		
		var prs []struct {
			Number    int       `json:"number"`
			Title     string    `json:"title"`
			URL       string    `json:"url"`
			State     string    `json:"state"`
			CreatedAt time.Time `json:"createdAt"`
			UpdatedAt time.Time `json:"updatedAt"`
		}
		
		if err := json.Unmarshal(output, &prs); err != nil {
			continue
		}
		
		// Filter by date and add to results
		for _, pr := range prs {
			if pr.CreatedAt.After(cutoffDate) || pr.UpdatedAt.After(cutoffDate) {
				allPRs = append(allPRs, PullRequest{
					Number:     pr.Number,
					Title:      pr.Title,
					URL:        pr.URL,
					Repository: repo,
					State:      pr.State,
					CreatedAt:  pr.CreatedAt,
					UpdatedAt:  pr.UpdatedAt,
				})
			}
		}
	}
	
	return allPRs, nil
}

func (c *Client) GetTodayPullRequests() ([]PullRequest, error) {
	return c.GetRecentPullRequests(1)
}
