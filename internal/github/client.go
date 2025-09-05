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
	cmd := exec.Command("gh", "repo", "list", "--limit", "20", "--json", "nameWithOwner")
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
	query := fmt.Sprintf(`
		query {
			repository(owner: "%s", name: "%s") {
				ref(qualifiedName: "refs/heads/main") {
					target {
						... on Commit {
							history(first: 100, author: {emails: ["%s"]}, since: "%sT00:00:00Z") {
								edges {
									node {
										oid
										message
										url
										authoredDate
									}
								}
							}
						}
					}
				}
			}
		}
	`, strings.Split(repo, "/")[0], strings.Split(repo, "/")[1], c.username, since)

	cmd := exec.Command("gh", "api", "graphql", "-f", fmt.Sprintf("query=%s", query))
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
		commits = append(commits, Commit{
			SHA:        edge.Node.OID,
			Message:    edge.Node.Message,
			URL:        edge.Node.URL,
			Repository: repo,
			AuthorDate: edge.Node.AuthoredDate,
		})
	}

	return commits, nil
}

func (c *Client) GetRecentPullRequests(days int) ([]PullRequest, error) {
	since := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	cmd := exec.Command("gh", "search", "prs",
		fmt.Sprintf("author:%s created:>=%s", c.username, since),
		"--json", "number,title,url,repository,state,createdAt,updatedAt",
		"--limit", "50")

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it's just no results (which is ok)
		if strings.Contains(string(output), "no pull requests found") {
			return []PullRequest{}, nil
		}
		return nil, fmt.Errorf("failed to search pull requests: %v - output: %s", err, string(output))
	}

	var searchResults []struct {
		Number     int    `json:"number"`
		Title      string `json:"title"`
		URL        string `json:"url"`
		Repository struct {
			NameWithOwner string `json:"nameWithOwner"`
		} `json:"repository"`
		State     string    `json:"state"`
		CreatedAt time.Time `json:"createdAt"`
		UpdatedAt time.Time `json:"updatedAt"`
	}

	// Handle empty results
	if len(output) == 0 || string(output) == "[]\n" || string(output) == "[]" {
		return []PullRequest{}, nil
	}

	if err := json.Unmarshal(output, &searchResults); err != nil {
		return nil, fmt.Errorf("failed to parse pull request results: %v", err)
	}

	var prs []PullRequest
	for _, sr := range searchResults {
		prs = append(prs, PullRequest{
			Number:     sr.Number,
			Title:      sr.Title,
			URL:        sr.URL,
			Repository: sr.Repository.NameWithOwner,
			State:      sr.State,
			CreatedAt:  sr.CreatedAt,
			UpdatedAt:  sr.UpdatedAt,
		})
	}

	return prs, nil
}

func (c *Client) GetTodayPullRequests() ([]PullRequest, error) {
	return c.GetRecentPullRequests(1)
}
