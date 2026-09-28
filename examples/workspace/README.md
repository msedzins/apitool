# Example workspace

Run the app from this directory and open the `users` collection:

```sh
cd examples/workspace
go run ../../cmd/apitool users
```

The tree contains the valid `users/list` request and the intentionally invalid
`broken` request. Move to `broken` and press Enter to see the missing
`request.url` diagnostic; the valid request remains available to open.
