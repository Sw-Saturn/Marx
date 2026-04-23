package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/go-github/v62/github"
)

type IssueCommentEvent struct {
	Action  string        `json:"action"`
	Comment *EventComment `json:"comment"`
	Issue   *EventIssue   `json:"issue"`
}

type EventComment struct {
	Body string `json:"body"`
}

type EventIssue struct {
	Number      int              `json:"number"`
	PullRequest *json.RawMessage `json:"pull_request"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("GITHUB_TOKEN is not set")
	}

	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	if eventPath == "" {
		return fmt.Errorf("GITHUB_EVENT_PATH is not set")
	}

	repo := os.Getenv("GITHUB_REPOSITORY")
	if repo == "" {
		return fmt.Errorf("GITHUB_REPOSITORY is not set")
	}

	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid GITHUB_REPOSITORY format: %s", repo)
	}
	owner, repoName := parts[0], parts[1]

	command := os.Getenv("INPUT_COMMAND")
	if command == "" {
		command = "/approve"
	}

	data, err := os.ReadFile(eventPath)
	if err != nil {
		return fmt.Errorf("reading event file: %w", err)
	}

	var event IssueCommentEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("parsing event: %w", err)
	}

	if event.Action != "created" {
		log.Println("skipping: event action is not 'created'")
		return nil
	}

	if event.Issue == nil || event.Issue.PullRequest == nil {
		log.Println("skipping: comment is not on a pull request")
		return nil
	}

	body := strings.TrimSpace(event.Comment.Body)
	if !matchCommand(body, command) {
		log.Printf("skipping: comment does not match command %q", command)
		return nil
	}

	ctx := context.Background()
	client := github.NewClient(nil).WithAuthToken(token)

	prNumber := event.Issue.Number
	log.Printf("approving PR #%d in %s/%s", prNumber, owner, repoName)

	_, _, err = client.PullRequests.CreateReview(ctx, owner, repoName, prNumber, &github.PullRequestReviewRequest{
		Event: github.String("APPROVE"),
	})
	if err != nil {
		return fmt.Errorf("creating review: %w", err)
	}

	log.Printf("PR #%d approved successfully", prNumber)
	return nil
}

func matchCommand(body, command string) bool {
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == command {
			return true
		}
	}
	return false
}
