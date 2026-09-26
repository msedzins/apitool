# Examples

The workspace fixture is used both for manual UI-001 checks and automated UI snapshots.

Run it locally:

```sh
cd /Users/msedzinski/github/apitool/examples/workspace
git init
go run ../../cmd/apitool
```

`git init` makes this directory the workspace root, so apitool discovers only the `payments` and `users` collections. The generated `.git/` and `.apitool/` directories remain untracked.
