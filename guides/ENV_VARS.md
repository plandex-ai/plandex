
# Environment Variables

Here is a list of all the environment variables (and their defaults) that you can use with `plandex`.

```bash
# --- General ---

OPENAI_API_BASE= # Your OpenAI server, such as http://localhost:1234/v1 Defaults to empty.
OPENAI_API_KEY= # Your OpenAI key.

# optional - set api keys for any other providers you're using
export OPENROUTER_API_KEY= # Your OpenRouter.ai API key.
export TOGETHER_API_KEY = # Your Together.ai API key.
# etc.

# --- Plandex specific ---

PLANDEX_DATA_DIR= # A directory to put plandex data files (cache etc.) to
PLANDEX_ENV= # Set this to development in order to use custom API host. Defaults to empty.

# Database
PGDATA_DIR=/var/lib/postgresql/data
DATABASE_URL=postgres://<user>:<password>@<host>:<port>/plandex?sslmode=disable
# or
PLANDEX_DB=plandex # Your postgres database.
PLANDEX_USER= # Your postgres user.
PLANDEX_PASSWORD= # Your postgres password.

# --- Development variables (See DEVELOPMENT.md) ---

PLANDEX_API_HOST= # Defaults to http://localhost:8080 if PLANDEX_ENV is development, otherwise it is http://api.plandex.ai
PLANDEX_OUT_DIR=/usr/local/bin # The output when using dev.sh

GOENV= # This should be set to development.
GOPATH= # This should be already set to your go folder if you've installed Go lang.

SMTP_HOST= # Youe SMTP host.
SMTP_PORT= # Set this to 1025 e.g. if you are using mailhog.
SMTP_USER= # SMTP username.
SMTP_PASSWORD= # SMTP password.

```