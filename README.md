# apitool

`apitool` is a terminal API client for request definitions kept in Git. It discovers collections inside the repository where it starts, lets you edit one YAML request at a time, and sends that request to the selected environment.

## Workspace layout

Each collection is a directory containing `.api/`:

```text
api-workspace/
├── .git/
├── .gitignore
└── payments/
    └── .api/
        ├── collection.yaml
        ├── environments/
        │   ├── test.yaml
        │   └── prod.yaml
        └── requests/
            └── users/list.yaml
```

Collection metadata, environment configuration, groups, and requests are reviewable YAML files under `.api/`. Add `.apitool/` to the workspace root `.gitignore`; apitool keeps response cache, request history, UI preferences, and redacted execution logs there. Runtime data stays local and is never staged by apitool.

```gitignore
.apitool/
```

## Safe environment and authentication configuration

Environment files hold non-secret values and may refer to process environment variables. Keep client credentials in the process environment:

```yaml
# payments/.api/environments/test.yaml
name: test
variables:
  base_url: https://api.example.test
  oauth_token_url: https://auth.example.test/oauth/token
```

```yaml
# payments/.api/collection.yaml
name: Payments API
auth:
  type: oauth2
  grant: client_credentials
  token_url: "{{oauth_token_url}}"
  client_id: ${PAYMENTS_CLIENT_ID}
  client_secret: ${PAYMENTS_CLIENT_SECRET}
  scopes:
    - payments.read
```

Set `PAYMENTS_CLIENT_ID` and `PAYMENTS_CLIENT_SECRET` in the shell or secret manager that starts apitool. Do not put credential values or access tokens in YAML. A request can inherit collection authentication, override it in a group, or set `auth: none` to disable it.

## Start apitool

Run apitool from inside the Git workspace. With no collection argument, it opens the collection picker. Use `-e` (or `--env`) to select the starting environment and `-c` (or `--confirm-dangerous`) to ask before sending POST, PUT, PATCH, and DELETE requests:

```sh
apitool
apitool -e test
apitool -e prod -c
```

Use the collection picker to select a collection, the environment picker to switch environments, `/` to search requests, and `e` to edit the selected request. The command palette (`Ctrl+K`) contains History and the Git actions. Press `?` for the keyboard guide.

## Editing and saving

Request edits are held in the editor until you save. `Ctrl+S` writes the current request definition. Leaving a dirty request asks whether to save, discard, or cancel. The editor changes only the selected request YAML file; it does not change environment or collection files unless you edit those separately.

## History and responses

History is stored in `.apitool/history.jsonl` and lists the newest executions first. It records safe metadata such as collection, environment, request ID, method, status or error category, and duration. It does not store request bodies, authorization headers, client secrets, or tokens. Selecting a history entry opens the current request definition at that path; history is not a saved request snapshot.

Successful HTTP responses are cached under `.apitool/responses/` by collection, environment, and request. The response viewer shows status, duration, size, redacted headers, and the body. HTTP error statuses are responses; transport and OAuth failures appear as diagnostics.

## Git actions and limits

The palette exposes Git Status, Diff, Pull, Push, and Commit. These call the local Git executable at the workspace root. Commit requires a nonblank message and commits changes that you have already staged outside apitool. Apitool never adds or stages files. Its Git surface does not manage branches, checkout, merge or rebase, conflict resolution, or remote provider settings. Runtime data under `.apitool/` is excluded from status and diff output and cannot be committed through the apitool adapter.

## Outside the MVP

The MVP runs one request at a time. It does not include collection synchronization, team accounts, shared cloud storage, request chaining, batch or parallel execution, retries, multipart or file bodies, mTLS, client certificates, or OAuth authorization-code login.

## Fixture workspace and local verification

`testdata/workspace/` is a small two-collection workspace used by the end-to-end app test. It includes an intentionally malformed request so the collection tree can demonstrate a visible validation failure while valid sibling requests remain usable. The test copies the fixture into a temporary directory, initializes a local Git repository, injects its endpoint and credentials through the process environment, and sends exactly one request to an in-process HTTP server. It checks response caching, history, redacted logs, and that `.apitool/` stays out of Git status. No external API or credential is required.
