# Mailboxsweep

[![Build and test Go package](https://github.com/nate-anderson/mailboxsweep/actions/workflows/ci.yml/badge.svg)](https://github.com/nate-anderson/mailboxsweep/actions/workflows/ci.yml)

## Running locally

Run locally with HTTPS using `local-ssl-proxy`:

1. `npm install -g local-ssl-proxy`
2. `mkdir localdev-certs`
3. `touch localdev-certs/config.json`
4. Paste the config below into `config.json`

```json
{
	"backend": {
		"source": 8081,
		"target": 8080,
		"cert": "localhost.pem",
		"key": "localhost-key.pem"
	},
	"frontend": {
		"source": 3001,
		"target": 3000,
		"cert": "localhost.pem",
		"key": "localhost-key.pem"
	}
}
```

5. Generate SSL certs
6. `local-ssl-proxy --config=config.json`

Spin up docker backend dependencies with `make up` in `./backend`

Migrate (create s3 bucket, Dynamo table, SQS queue) with:
`go run ./cmd/migrate s3`
`go run ./cmd/migrate sqs`
`go run ./cmd/migrate dynamo`

Run frontend with `npm run dev`

Run backend with `go run ./cmd/server`

## Authentication

Auth uses magic links. We store user IMAP credentials securely when they're not using a supported direct login provider (just Gmail right now). For Gmail users, we store their access token and refresh token for use with the Gmail API.

If a user logs in for the first time, we check their domain and redirect them to Google if applicable. Otherwise we go straight to magic link auth.

Returning users always use magic links to log in so we don't have to send them through the Google auth flow multiple times.

# Generating RSA signing certificate

Login links include an RSA signature for extra security. This requires a `.pem` key, which can be generated using:

- `ssh-keygen -t rsa -m PEM`

## Configuration (SSM)

All non-sensitive config is handled via environment variables. Sensitive configuration should
be stored as secure strings in SSM ParamStore. The SSM param store driver will check the `ENV`
environment variable and prefix SSM API calls accordingly:

`prod` => `/mailboxsweep/PROD_`
`staging` => `/mailboxsweep/STAGING_`
anything else => no prefix

## Deploying Infrastructure Changes

The `infra` directory is an AWS CDK project (Typescript) defining the project's infrastructure on AWS.

`MAILBOXSWEEP_ENV=prod AWS_PROFILE=mailboxsweep cdk deploy InfraStack/mailboxsweep-prod`

## Deploying the frontend

```bash
cd web
npm run build:staging
npm run robots:staging
cd out
aws s3 sync . s3://com.mailboxsweep.staging.web --delete
```

This should be automated in CD once branch protection is figured out.

## Maddy setup

Download default 0.7 config file: `make maddy_init_config`

Change the line beginning with `tls` to look like this:

```conf
tls file /data/tls/fullchain.pem /data/tls/privkey.pem
```

Create TLS certs with: `make maddy_certs`

Move them to `.docker-dev/volumes/maddy/tls`

Create the credentials used in CI unit tests and browser tests: `make maddy_creds`

This sets the username to `testuser@mailboxsweep.local` and the password to `pass123`.

## Server configuration

The server binary is located at `backend/cmd/server`. The majority of the server's configuration
comes from environment variables (and `.env` files), but in deployed environments, sensitive
config variables are stored in AWS SSM Paramstore. The server first checks the environment then
SSM for params, and errors if a required param is found in neither source.

A few flags exist for bootstrapping the application:

- `--envFile`: specify the location of the configuration file in `.env` format, defaults to `./.env`
- `--ssmHost`: specify the host of the SSM Paramstore API, useful when you want to use localstack instead of real AWS
- `--ssmPort`: specify the port of the SSM Paramstore API, see above
