# Card Onboarding Worker

## Responsibility

`card-onboarding-worker` is an AWS Lambda worker that consumes accepted CSV row messages from SQS, validates business fields, masks card numbers in logs, and calls `onboard-service` through the generated OpenAPI client.

## Architecture / Flow

```text
card-onboarding-worker-sqs-{env}
  -> card-onboarding-worker Lambda
  -> parse onboarding message
  -> validate business fields
  -> call onboard-service POST /internal/cards/onboard
  -> let SQS retry technical failures
```

The worker does not download the original CSV file from S3. It processes one accepted record message at a time.

## API List

Not applicable. This component is an SQS-triggered Lambda worker.

## Swagger Location

Not applicable for the worker. The service it calls is defined at `../card-onboarding-services/onboard-service/swagger-internal.yaml`.

## Generated Client Package Location

`../card-onboarding-services/onboard-service/pkg/onboard`

## Config Variables

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `ONBOARD_SERVICE_BASE_URL` | Yes | none | Base URL for `onboard-service`. |
| `ONBOARD_SERVICE_TIMEOUT` | No | `5s` | Go duration for onboard-service HTTP calls. |

## Queue/DLQ Names

- Queue: `card-onboarding-worker-sqs-{env}`
- DLQ: `card-onboarding-worker-dlq-{env}`

## Retry Rules

SQS owns retry:

- Message parse failure: return error, retry, then DLQ after `3` receives.
- Business validation failure: log and return success; no retry and no DLQ.
- `onboard-service` `2xx`: return success.
- `onboard-service` `4xx`: return success; no retry and no DLQ.
- `onboard-service` `5xx`, timeout, or network error: return error, retry, then DLQ after `3` receives.

Queue settings:

- visibility timeout: `60 seconds`
- message retention: `4 days`
- DLQ max receive count: `3`

## Local Run Command

This Lambda is normally run by AWS Lambda. For local compilation from `card-onboarding-workers`:

```sh
go build ./card-onboarding-worker
```

## Docker Build Command

Not applicable. Workers are packaged as Lambda zip artifacts:

```sh
make lambda-package
```

## Unit Test Command

From `card-onboarding-workers`:

```sh
go test ./card-onboarding-worker/...
```

## Smoke Test Command

From `card-onboarding-workers`:

```sh
make smoke-test
```

Full AWS smoke tests require `SMOKE_TEST_ENABLED=true` and the `SMOKE_*` variables documented in `smoke-test/config.go`.

## Deployment Command

From `card-onboarding-workers`:

```sh
make deploy-test
```

For production:

```sh
make deploy-prod ONBOARD_SERVICE_BASE_URL=https://example.internal
```
