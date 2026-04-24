# Slash Approve

A GitHub Action that approves a pull request when a designated slash command
(default `/approve`) is posted as a PR comment.

## How it works

The action is triggered by the `issue_comment` event. When the comment matches
the configured command and is on a pull request, the action creates an approving
review on that PR via the GitHub REST API.

## Requirements

- The workflow file **must live on the repository's default branch** — GitHub
  only dispatches `issue_comment` events to workflows defined on the default
  branch.
- A `github-token` input that can submit a review on the target PR. Two
  independent rules govern the choice of token:

  1. **Reviewer identity must differ from the PR author.** GitHub rejects a
     review with `422 Unprocessable Entity: Review Can not approve your own
     pull request` when the reviewer is the same user as the PR author. This
     blocks self-approval with a PAT owned by the PR author.
  2. **If the target repo enforces a required approving review count** (via
     branch protection or a ruleset), the reviewer must be recognized as
     having **write access to the repository** for the review to count. An
     approving review from an identity without write access is recorded on
     the PR but leaves `reviewDecision` at `REVIEW_REQUIRED`, so the PR stays
     blocked.

| Token | Can self-approve? | Satisfies required-review rules? | Setup |
|---|---|---|---|
| `GITHUB_TOKEN` | Only when the PR author is not `github-actions[bot]` (typical) | Not reliable — treated as a bot review, often does not satisfy required counts | Enable "Allow GitHub Actions to create and approve pull requests" in repo Actions settings, plus `pull-requests: write` on the job |
| PAT (fine-grained) | Blocked when the PAT owner is the PR author | Yes, as long as the PAT owner has repo write access | Fine-grained PAT with `Pull requests: Read and write` |
| PAT (classic) | Blocked when the PAT owner is the PR author | Yes, as long as the PAT owner has repo write access | Classic PAT with `repo` scope |
| **GitHub App installation token** (recommended) | Works — the app acts as a distinct bot identity | Only when the app has `Contents: Read and write` **in addition to** `Pull requests: Read and write`. With `Pull requests` alone, the review is submitted but does not count toward required reviews. | See [Quickstart (GitHub App)](#quickstart-github-app) |

## Quickstart (GitHub App)

This is the only path that supports approving PRs authored by the same user
who triggered the comment. Use it for solo repositories and automation.

1. Create a GitHub App: Settings → Developer settings → GitHub Apps → **New GitHub App**.
   - Webhook: **uncheck Active** (not needed).
   - Repository permissions:
     - **Pull requests: Read and write** — required to submit reviews.
     - **Contents: Read and write** — required for the review to count toward
       required-review rules (branch protection / rulesets). Grant this unless
       you are certain the target repo does not enforce such a rule; without
       it the app's approval is submitted but does not satisfy the required
       count and the PR stays blocked.
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
      - uses: Sw-Saturn/Marx@v2
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
      - uses: Sw-Saturn/Marx@v2
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
uses: Sw-Saturn/Marx@v2
```

`v2` and newer is a composite action with no container build overhead.
`v1.x` (Docker-based) remains available for consumers that have pinned to it,
but is no longer developed.

## Runner requirements

The composite action shells out to `jq` and `gh`. Both are pre-installed on
GitHub-hosted `ubuntu-*` runners, so no setup is needed there. Self-hosted
runners need them on `PATH`.

## Development

Nothing to build — the action is a single shell step in `action.yml`.

### Run the action locally against a fixture

Stage one of the fixtures in `testdata/` as the event payload and invoke the
shell step with a dummy token:

```sh
export GITHUB_EVENT_PATH="$(mktemp)"
cp testdata/issue_comment_approve.json "$GITHUB_EVENT_PATH"
export GITHUB_REPOSITORY=test/repo
export GH_TOKEN=dummy-token
export INPUT_COMMAND=/approve

# Extract and run the shell body of the composite action
yq '.runs.steps[0].run' action.yml | bash
```

For the approve fixture a `401 Bad credentials` response from `api.github.com`
after the `approving PR #1 in test/repo` log line is the expected "reached the
API" outcome. For `issue_comment_no_match.json` /
`issue_comment_not_pr.json`, expect a `skipping: ...` log and exit 0.

### CI

The `Test` workflow runs the composite action against each fixture in
`testdata/` and asserts the resulting step outcome:

- `no_match` / `not_pr` → expected success (action exits 0 after the skip log)
- `approve` → expected failure (dummy token → 401 from the API)

## Privacy

See [PRIVACY.md](./PRIVACY.md).

## License

[MIT](./LICENSE)
