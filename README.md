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

## Configuring Greenmail to work with local SSL

Greenmail is an SMTP/IMAP server bundle used for testing.

https://github.com/greenmail-mail-test/greenmail/issues/448#issuecomment-1068376521

1. Generate cert

```
openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
 -keyout greenmail.key -out greenmail.crt -subj "/CN=localhost" \
 -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
```

2. Generate .p12 keystore

```
openssl pkcs12 -export -out greenmail.p12 -inkey greenmail.key -in greenmail.crt
```

When it asks for a password, use `changeit`. 3. Add certificate as a Trusted Root Certificate Authority

```
sudo cp greenmail.crt /usr/local/share/ca-certificates/

# arch
# https://wiki.archlinux.org/title/User:Grawity/Adding_a_trusted_CA_certificate
sudo trust anchor --store ./greenmail.crt

sudo update-ca-certificates
```

4. Add the keystore to the JAR (located at https://repo1.maven.org/maven2/com/icegreen/greenmail-standalone/1.6.11/greenmail-standalone-1.6.11.jar)

```
jar uvf greenmail-standalone.jar greenmail.p12
```

5. Bind mount when running docker

```
docker run -v <full path to greenmail-standalone.jar>:/home/greenmail/greenmail-standalone.jar -t -i -e GREENMAIL_OPTS='-Dgreenmail.setup.test.all -Dgreenmail.hostname=0.0.0.0 -Dgreenmail.auth.disabled -D-Dgreenmail.verbose' -p 3025:3025 -p 3110:3110 -p 3143:3143 -p 3465:3465 -p 3993:3993 -p 3995:3995 greenmail/standalone:1.6.7
```

Using the full path of the local JAR file to get it working

# I didn't actually try any of this yet ^^^

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
