# Card Onboarding File Preprocessor

## Responsibility

`card-onboarding-file-preprocessor` is an AWS Lambda worker that consumes S3 object-created events from SQS, downloads uploaded CSV files, validates file and row structure, writes preprocessing results to S3, and publishes structurally accepted records to the onboarding worker queue.

## Architecture / Flow

```text
S3 input bucket
  -> S3 ObjectCreated event
  -> card-onboarding-file-preprocessor-sqs-{env}
  -> card-onboarding-file-preprocessor Lambda
  -> download source CSV
  -> validate extension, size, header, and row column counts
  -> write result CSV to S3 output bucket
  -> publish accepted rows to card-onboarding-worker-sqs-{env}
```

## API List

Not applicable. This component is an SQS-triggered Lambda worker.

## Swagger Location

Not applicable.

## Generated Client Package Location

Not applicable. This worker uses S3 and SQS clients, not a generated OpenAPI client.

## Config Variables

| Variable | Required | Description |
| --- | --- | --- |
| `OUTPUT_BUCKET_NAME` | Yes | S3 bucket for preprocessing result CSV files. |
| `MAX_FILE_SIZE_BYTES` | Yes | Maximum accepted CSV file size in bytes. |
| `WORKER_QUEUE_URL` | Yes | SQS queue URL for accepted onboarding records. |

## Queue/DLQ Names

- Queue: `card-onboarding-file-preprocessor-sqs-{env}`
- DLQ: `card-onboarding-file-preprocessor-dlq-{env}`

## Retry Rules

SQS owns retry. Technical failures return an error and are retried until the queue's `maxReceiveCount` of `3`, then moved to the DLQ. Examples include invalid SQS/S3 event parsing, S3 download failure, file-level validation failure, result upload failure, and SQS publish failure.

Queue settings:

- visibility timeout: `60 seconds`
- message retention: `4 days`
- DLQ max receive count: `3`

Malformed CSV rows are written to the preprocessing result file and are not published to the worker queue.

## Local Run Command

This Lambda is normally run by AWS Lambda. For local compilation from `card-onboarding-workers`:

```sh
go build ./card-onboarding-file-preprocessor
```

## Docker Build Command

Not applicable. Workers are packaged as Lambda zip artifacts:

```sh
make lambda-package
```

## Unit Test Command

From `card-onboarding-workers`:

```sh
go test ./card-onboarding-file-preprocessor/...
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
