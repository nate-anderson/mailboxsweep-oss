# Mailboxsweep

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
