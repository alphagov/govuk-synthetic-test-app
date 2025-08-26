package helpers

import (
	"context"
	"time"

	"github.com/google/go-github/v74/github"
)

func CommitVersionChange(ctx context.Context, client *github.Client, date time.Time, tagToPush string, author *github.CommitAuthor) error {
	entries := []*github.TreeEntry{}
	entries = append(entries, &github.TreeEntry{Path: github.Ptr(".version"), Type: github.Ptr("blob"), Content: github.Ptr(string(tagToPush)), Mode: github.Ptr("100644")})
	mainRef, _, err := client.Git.GetRef(ctx, "alphagov", "govuk-synthetic-test-app", "refs/heads/main")
	if err != nil {
		return err
	}

	newBranch, _, createRefErr := client.Git.CreateRef(ctx, "alphagov", "govuk-synthetic-test-app", &github.Reference{Ref: github.Ptr("refs/heads/" + tagToPush), Object: &github.GitObject{Type: github.Ptr(*mainRef.Object.Type), SHA: github.Ptr(*mainRef.Object.SHA)}})
	if createRefErr != nil {
		return err
	}

	tree, _, err := client.Git.CreateTree(ctx, "alphagov", "govuk-synthetic-test-app", *newBranch.Object.SHA, entries)
	if err != nil {
		return err
	}

	newBranchParent, _, err := client.Repositories.GetCommit(ctx, "alphagov", "govuk-synthetic-test-app", *newBranch.Object.SHA, nil)
	if err != nil {
		return err
	}

	newBranchParent.Commit.SHA = newBranchParent.SHA
	commit := github.Commit{Author: author, Message: github.Ptr("sythentic deployment test: " + tagToPush), Tree: tree, Parents: []*github.Commit{newBranchParent.Commit}}
	opts := github.CreateCommitOptions{}
	createdCommit, _, commitErr := client.Git.CreateCommit(ctx, "alphagov", "govuk-synthetic-test-app", &commit, &opts)
	if commitErr != nil {
		return commitErr
	}

	newBranch.Object.SHA = createdCommit.SHA
	_, _, updareRefErr := client.Git.UpdateRef(ctx, "alphagov", "govuk-synthetic-test-app", newBranch, false)
	if updareRefErr != nil {
		return updareRefErr
	}

	branchToMerge := github.RepositoryMergeRequest{
		Base:          github.Ptr("main"),
		Head:          newBranch.Object.SHA,
		CommitMessage: github.Ptr("Synthetic test: Merging " + tagToPush + " into main"),
	}

	_, _, mergeErr := client.Repositories.Merge(ctx, "alphagov", "govuk-synthetic-test-app", &branchToMerge)
	if mergeErr != nil {
		return mergeErr
	}

	client.Git.DeleteRef(ctx, "alphagov", "govuk-synthetic-test-app", "refs/heads/"+tagToPush)

	return nil
}

func GetCurrentRelease(ctx context.Context, client *github.Client) (*github.RepositoryRelease, error) {
	latestRelease, _, err := client.Repositories.GetLatestRelease(ctx, "alphagov", "govuk-synthetic-test-app")
	if err != nil {
		return nil, err
	}
	return latestRelease, nil
}
