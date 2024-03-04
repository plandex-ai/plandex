## Development

To set up a development environment, first install dependencies:

- Go 1.21.3 - [install here](https://go.dev/doc/install)
- [reflex](https://github.com/cespare/reflex) 0.3.1 - for watching files and rebuilding in development. Install with `go install github.com/cespare/reflex@v0.3.1`

Then from the root directory, run:

```bash
./dev.sh
```

This creates watchers with `reflex` to rebuild both the server and the CLI when relevant files change.

The server runs on port 8088 by default.

After each build, the CLI is copied to `/usr/local/bin/plandex` so you can use it with just `plandex` in any directory. A `pdx` alias is also created.
