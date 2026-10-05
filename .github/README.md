# Repository automation

## Local CI toolchain

`.mise.toml` pins Go to `1.26.0`, the minimum version declared by `go.work` and
all module files. It sets `GOTOOLCHAIN=local` so dependency or tooling changes
cannot silently select a newer Go version. GitHub CI also selects Go from
`go.work`. Keep the mise pin aligned when intentionally changing the minimum.

Run `mise exec -- go version` to confirm the selected toolchain, then `mise ci`
for the full local checks.

## Dependency PR GitHub App

The `Update Submodule Dependencies` workflow updates the Go submodules to reference
main and opens or updates `bot/update-submodule-dependencies`. It authenticates as
**Speakeasy OpenAPI Dependencies** so the resulting PR checks run automatically.
PRs created or updated with `GITHUB_TOKEN` require manual workflow approval.

The App is an authentication identity only. The implementation lives in
[`workflows/update-submodule-dependencies.yaml`](workflows/update-submodule-dependencies.yaml);
there is no separately hosted service, webhook handler, or OAuth login flow.

### Ownership and access

- Owner: `speakeasy-api` organisation. Organisation owners manage the App.
- [App settings](https://github.com/organizations/speakeasy-api/settings/apps/speakeasy-openapi-dependencies)
- [Installation settings](https://github.com/organizations/speakeasy-api/settings/installations/168310303)
- App ID: `5203735`.
- Client ID: `Iv23lix0zkHJ35Cthk16` (public identifier, not a secret).
- Private App, installable only in `speakeasy-api`.
- Installation access: only `speakeasy-api/openapi`.
- Repository permissions: Contents and Pull requests read/write, plus mandatory
  Metadata read-only. No organisation permissions, Actions administration, or
  permission to modify workflow files.
- Webhooks and user authorisation are disabled.

### Repository configuration

Configure these under the repository's **Settings > Secrets and variables > Actions**:

| Type     | Name                  | Value                               |
| -------- | --------------------- | ----------------------------------- |
| Variable | `BOT_APP_CLIENT_ID`   | `Iv23lix0zkHJ35Cthk16`              |
| Secret   | `BOT_APP_PRIVATE_KEY` | The App's generated PEM private key |

The workflow generates an installation token only when dependencies change. It
explicitly restricts the token to this repository and requests only Contents and
Pull requests write access. `actions/create-github-app-token` revokes the token at
job completion; installation tokens also expire after one hour. The normal
`GITHUB_TOKEN` has read-only Contents access.

Do not put the private key in Git, documentation, logs, or PR comments. It is stored
as an encrypted repository Actions secret. No local copy is retained after setup.
Do not expose it to workflows that execute untrusted PR code.

### Rotate the private key

1. Generate a new private key under **App settings > General > Private keys**.
2. Replace the `BOT_APP_PRIVATE_KEY` repository Actions secret. For example:
   `gh secret set BOT_APP_PRIVATE_KEY --repo speakeasy-api/openapi < /secure/path/new-key.pem`.
3. Confirm the next dependency-update run can generate its token and create or
   update a PR whose checks start without approval.
4. Delete the old key from the App settings, then remove the downloaded new key
   from local storage. GitHub cannot show the stored Actions secret again; generate
   another key if it needs replacing.

### Disable or remove

Suspend or uninstall the App through its installation settings to revoke access.
For permanent removal, also remove `BOT_APP_CLIENT_ID` and `BOT_APP_PRIVATE_KEY`
and update or disable the dependency-update workflow. Reverting to `GITHUB_TOKEN`
restores manual approval for the generated PR workflows.

Fork workflow-approval settings are unchanged. This App does not bypass branch
protection, required checks, review requirements, or merge queues.
