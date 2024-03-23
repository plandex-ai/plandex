# Plandex development 🛠️

To set up a development environment, first install dependencies:

- Go 1.21.3 - [install here](https://go.dev/doc/install)
- [reflex](https://github.com/cespare/reflex) 0.3.1 - for watching files and rebuilding in development. Install with `go install github.com/cespare/reflex@v0.3.1`
- PostreSQL 14 - https://www.postgresql.org/download/

Make sure the PostgreSQL server is running and create a database called `plandex`.

Then make sure the following environment variables are set:

```bash
export DATABASE_URL=postgres://user:password@host:5432/plandex # replace with your own database URL
export GOENV=development
```

Note: [EnvKey](https://www.envkey.com/) is a good way to manage environment variables in development.

Now from the root directory of this repo, run:

```bash
./dev.sh
```

This creates watchers with `reflex` to rebuild both the server and the CLI when relevant files change.

The server runs on port 8088 by default.

After each build, the CLI is copied to `/usr/local/bin/plandex` so you can use it with just `plandex` in any directory. A `pdx` alias is also created.

When running the Plandex CLI, set `export PLANDEX_ENV=development` to run in development mode, which connects to the development server by default.
