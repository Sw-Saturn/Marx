# Slash Approve

A GitHub Action that approves a pull request when a designated slash command
(default `/approve`) is posted as a PR comment.

## How it works

The action is triggered by the `issue_comment` event. When the comment matches
the configured command and is on a pull request, the action creates an approving
review on that PR via the GitHub REST API.

## Requirements

- The workflow file **must live on the repository's default branch**. GitHub only
  dispatches `issue_comment` events to workflows defined on the default branch.
- A token with `pull_requests: write` permission on the target repository. The
  built-in `GITHUB_TOKEN` cannot approve a PR authored by the same user, so
  self-approval requires one of:
  - a **Personal Access Token** (fine-grained with `Pull requests: Read and write`,
    or classic with `repo` scope), or
  - a **GitHub App installation token**.

## Quickstart

1. Generate a token (fine-grained PAT is the simplest):
   - Settings → Developer settings → Personal access tokens → Fine-grained tokens
   - Repository access: only the target repositories
   - Repository permissions → **Pull requests: Read and write**
2. Add the token as a repository secret named `APPROVE_TOKEN`
   (Settings → Secrets and variables → Actions → New repository secret).
3. Add the following workflow to your default branch:

```yaml
# .github/workflows/approve.yml
name: Slash Approve
on:
  issue_comment:
    types: [created]

jobs:
  approve:
    if: |
      github.event.issue.pull_request &&
      contains(github.event.comment.body, '/approve') &&
      (github.event.comment.author_association == 'OWNER' ||
       github.event.comment.author_association == 'MEMBER' ||
       github.event.comment.author_association == 'COLLABORATOR')
    runs-on: ubuntu-latest
    steps:
      - uses: Sw-Saturn/Marx@main
        with:
          github-token: ${{ secrets.APPROVE_TOKEN }}
```

4. Open a PR, post `/approve`, and the action will create an approving review.

## Inputs

| Input | Required | Default | Description |
|---|---|---|---|
| `github-token` | yes | — | Token used to create the PR review. |
| `command` | no | `/approve` | Slash command that triggers approval. Must match on a line by itself (leading/trailing whitespace ignored). |

## Authorization

The workflow snippet above includes an `author_association` check so only
repository members can trigger approval. **Do not omit this guard on public
repositories** — without it, anyone who can comment (i.e. any GitHub user) can
approve their own PR.

Recognized `author_association` values include `OWNER`, `MEMBER`,
`COLLABORATOR`, and `CONTRIBUTOR`. Pick the set that matches your trust model.

## Notes on `issue_comment`

- The workflow runs against the repository's default branch, not the PR branch.
  If you extend this action to inspect PR contents, pass
  `ref: refs/pull/${{ github.event.issue.number }}/merge` to `actions/checkout`.
- Only pull request comments are handled; issue comments are filtered out by
  `github.event.issue.pull_request`.

## Versioning

Pin to a released tag in production, not `@main`:

```yaml
uses: Sw-Saturn/Marx@v1.0.0
```

## Development

### Run unit tests

```sh
go test ./...
```

### Reproduce the CI simulation locally

`act` is used to replay `issue_comment` payloads against the production
workflow. Fixture events live under `testdata/`.

```sh
act issue_comment \
  -P ubuntu-latest=catthehacker/ubuntu:act-latest \
  -e testdata/issue_comment_approve.json \
  -s APPROVE_TOKEN=dummy-token
```

On Rancher Desktop / Apple Silicon, additionally pass
`DOCKER_HOST=unix://$HOME/.rd/docker.sock` and
`--container-daemon-socket /var/run/docker.sock --container-architecture linux/arm64`.

### CI

The `Test` workflow runs on every pull request:

- `go-test` — `go test -v ./...`
- `docker-build` — builds the action image with full build output
- `act-simulate` — matrix across the three fixture events, asserting exit code
  and expected log lines

## Privacy

See [PRIVACY.md](./PRIVACY.md).
