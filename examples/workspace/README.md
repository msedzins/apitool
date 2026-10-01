# Example workspace

From the repository root, open the `users` collection in this example workspace:

```sh
go run ./cmd/apitool -e test examples/workspace/users
```

The tree contains the valid `users/list` request and the intentionally invalid
`broken` request. Move to `broken` and press Enter to see the missing
`request.url` diagnostic; the valid request remains available to open.
