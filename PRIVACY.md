# Privacy Policy

Last updated: 2026-04-23

## Summary

Slash Approve does not collect, store, or share user data outside the GitHub
Actions environment it runs in. It does not send data to any third-party
service. The only network destination it contacts is `api.github.com`.

## Data the action processes

When triggered, the action reads the following from the GitHub-provided event
payload (`GITHUB_EVENT_PATH`) and environment (`GITHUB_REPOSITORY`):

- The pull request / issue number
- Whether the comment is on a pull request
- The comment body (to match against the configured slash command)
- The repository owner and name

All of these values are already available to any workflow run in the same
repository. The action does not persist them anywhere.

## Data sent over the network

The action makes a single authenticated HTTPS call to the GitHub REST API:

- `POST https://api.github.com/repos/{owner}/{repo}/pulls/{number}/reviews`
  with `event: "APPROVE"`.

No other outbound requests are made. The `github-token` value supplied via
the `github-token` input is used only as the `Authorization` header on this
request.

## Logs

Standard output and errors emitted by the action (approval result, skip
messages, API errors) are written to the GitHub Actions runner's log stream
and stored by GitHub under its own retention policy. This project does not
collect or forward those logs elsewhere.

## Third parties

None. No analytics, telemetry, or external SDKs are embedded in the action.

## Source

The full source of the action is available at
<https://github.com/Sw-Saturn/Marx>. Review `main.go` and the `Dockerfile` to
verify the behavior described above.

## Contact

Report concerns or request clarification via
<https://github.com/Sw-Saturn/Marx/issues>.
