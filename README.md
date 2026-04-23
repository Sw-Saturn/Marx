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
- A token that can create PR reviews on the target repository. **The identity
  behind the token must differ from the PR author** — GitHub rejects a review if
  the reviewer is the same user as the PR author with
  `422 Unprocessable Entity: Can not approve your own pull request`. This rules
  out self-approval with a PAT owned by the PR author.

| Scenario | Token type |
|---|---|
| Approve PRs authored by other users | Fine-grained PAT with `Pull requests: Read and write`, or classic PAT with `repo` scope |
| Approve your own PRs (solo project, automation) | **GitHub App installation token** (the app acts as a distinct bot identity) |

The built-in `GITHUB_TOKEN` cannot approve PRs in either case by default.

## Quickstart (GitHub App)

This is the only path that supports approving PRs authored by the same user
who triggered the comment. Use it for solo repositories and automation.

1. Create a GitHub App: Settings → Developer settings → GitHub Apps → **New GitHub App**.
   - Webhook: **uncheck Active** (not needed).
   - Repository permissions → **Pull requests: Read and write**.
2. Generate and download a private key (`.pem`) from the app's settings page.
3. Install the app on the target repository.
4. Add two repository secrets:
   - `MARX_APP_ID` — the app's numeric ID (shown on the app's settings page).
   - `MARX_APP_PRIVATE_KEY` — the full contents of the `.pem` file.
5. Add the following workflow to the default branch:

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
      - uses: actions/checkout@v4
      - uses: actions/create-github-app-token@v1
        id: app-token
        with:
          app-id: ${{ secrets.MARX_APP_ID }}
          private-key: ${{ secrets.MARX_APP_PRIVATE_KEY }}
      - uses: Sw-Saturn/Marx@main
        with:
          github-token: ${{ steps.app-token.outputs.token }}
```

6. Open a PR, post `/approve`, and the action will create an approving review
   under the app's bot identity.

## Quickstart (PAT, non-self-approval only)

Use a PAT when you only need to approve PRs authored by **other** users.
Approving your own PR with a PAT you own will be rejected by GitHub.

1. Generate a fine-grained PAT:
   - Settings → Developer settings → Personal access tokens → Fine-grained tokens.
   - Repository access: only the target repositories.
   - Repository permissions → **Pull requests: Read and write**.
2. Add the token as a repository secret named `APPROVE_TOKEN`.
3. Use this workflow on the default branch:

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

Two complementary test paths are used locally, mirroring CI:

1. `act` replays `issue_comment` payloads against the production workflow to
   verify the `if:` guard filters correctly. Fixture events live under
   `testdata/`. The production workflow depends on
   `actions/create-github-app-token`, so the approve fixture is exercised
   separately (see step 2) rather than through act.

   ```sh
   act issue_comment \
     -P ubuntu-latest=catthehacker/ubuntu:act-latest \
     -e testdata/issue_comment_no_match.json \
     -s APPROVE_TOKEN=dummy-token
   ```

   On Rancher Desktop / Apple Silicon, additionally pass
   `DOCKER_HOST=unix://$HOME/.rd/docker.sock` and
   `--container-daemon-socket /var/run/docker.sock --container-architecture linux/arm64`.

2. Run the built image directly against the approve fixture to verify the
   binary reaches the API call:

   ```sh
   docker build -t slash-approve .
   docker run --rm \
     -e GITHUB_TOKEN=dummy-token \
     -e GITHUB_REPOSITORY=test/repo \
     -e GITHUB_EVENT_PATH=/event.json \
     -v "$PWD/testdata/issue_comment_approve.json:/event.json:ro" \
     slash-approve
   ```

   With a dummy token the API call returns `401` / `422` — the log line
   `approving PR #1 in test/repo` printed just before the error proves the
   filter and event parsing are working.

### CI

The `Test` workflow runs on every pull request:

- `go-test` — `go test -v ./...`
- `docker-build` — builds the action image and runs the binary against the
  approve fixture end-to-end
- `act-simulate` — replays the `no_match` and `not_pr` fixtures through the
  production workflow to verify the `if:` guard

## Privacy

See [PRIVACY.md](./PRIVACY.md).

## License

[MIT](./LICENSE)
