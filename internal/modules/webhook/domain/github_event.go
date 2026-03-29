package domain

import (
	"strings"
	"time"
)

// GitHubPushEvent represents the relevant fields from a GitHub push webhook payload.
type GitHubPushEvent struct {
	Ref        string       `json:"ref"`
	Before     string       `json:"before"`
	After      string       `json:"after"`
	Compare    string       `json:"compare"`
	Commits    []Commit     `json:"commits"`
	HeadCommit *Commit      `json:"head_commit"`
	Pusher     GitHubUser   `json:"pusher"`
	Sender     GitHubSender `json:"sender"`
	Repository Repository   `json:"repository"`
}

// Commit represents a single commit in a push event.
type Commit struct {
	ID        string    `json:"id"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	URL       string    `json:"url"`
	Author    Author    `json:"author"`
}

// Author represents a commit author.
type Author struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

// GitHubUser represents the pusher info.
type GitHubUser struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// GitHubSender represents the sender (has avatar).
type GitHubSender struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
}

// Repository represents the repository info from the payload.
type Repository struct {
	FullName string `json:"full_name"`
	HTMLURL  string `json:"html_url"`
}

// ShortSHA returns first 7 chars of a commit SHA.
func ShortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// BranchName extracts branch name from refs/heads/xxx.
func (e *GitHubPushEvent) BranchName() string {
	const prefix = "refs/heads/"
	if strings.HasPrefix(e.Ref, prefix) {
		return e.Ref[len(prefix):]
	}
	return e.Ref
}
