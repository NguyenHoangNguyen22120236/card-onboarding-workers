# Card Onboarding Workers

Event-driven Go workers for the Card Onboarding File Processing Platform.

This repository contains two AWS Lambda workers:

- `card-onboarding-file-preprocessor`: consumes S3 object-created events from SQS, downloads uploaded CSV files, validates file structure, writes preprocessing results to S3, and publishes accepted rows to the worker queue.
- `card-onboarding-worker`: consumes accepted row messages from SQS, validates business fields, masks card numbers in logs, and calls the onboard service.

## Architecture

```text
Manual CSV Upload
  -> S3 input bucket
  -> S3 ObjectCreated event
  -> card-onboarding-file-preprocessor-sqs-{env}
  -> card-onboarding-file-preprocessor Lambda
  -> S3 output bucket
  -> card-onboarding-worker-sqs-{env}
  -> card-onboarding-worker Lambda
  -> onboard-service
```

The preprocessor is the only component that downloads the original CSV file from S3. The onboarding worker only receives one JSON message per structurally accepted CSV row.

## Repository Layout

```text
card-onboarding-file-preprocessor/  S3 event, CSV parsing, preprocessing result, S3/SQS clients
card-onboarding-worker/             Business validation and onboard-service client
docs/                               Architecture, processing flow, retry, and DLQ notes
infra/                              AWS CDK v2 stack written in Go
scripts/                            Lambda packaging scripts
Makefile                            Build, test, package, and synth shortcuts
```

## CSV Input Contract

Uploaded files must be `.csv` files and must use this exact header:

```csv
customer_id,card_type,card_number,expiry_date,holder_name,email
```

Example:

```csv
customer_id,card_type,card_number,expiry_date,holder_name,email
CUST001,VISA,4111111111111111,12/28,Nguyen Van A,a@example.com
CUST002,MASTERCARD,5555555555554444,10/27,Tran Van B,b@example.com
```

The preprocessor validates:

- file extension is `.csv`
- file size is greater than `0`
- file size does not exceed `MAX_FILE_SIZE_BYTES`
- file has the required header row
- file has at least one data row
- each data row has exactly 6 columns

Rows that pass structure validation are published to the worker queue. Malformed rows are recorded in the preprocessing result CSV and are not published.

## Preprocessing Output

For each processed file, the preprocessor writes a result CSV to:

```text
s3://card-onboarding-output-bucket-{env}/processed/{jobId}/{sourceFileName}_preprocess_result.csv
```

Output format:

```csv
record_id,row_number,customer_id,preprocess_status,error_message
REC-001,2,CUST001,ACCEPTED,
,3,CUST002,REJECTED,csv row has 4 columns, expected 6
```

Accepted records receive IDs like `REC-001`, `REC-002`, and so on. The generated job ID is also used as the correlation ID for messages emitted by the preprocessor.

## Worker Message Contract

Accepted rows are published to `card-onboarding-worker-sqs-{env}` as JSON:

```json
{
  "correlationId": "generated-job-id",
  "jobId": "generated-job-id",
  "recordId": "REC-001",
  "sourceFile": "cards_20260606.csv",
  "rowNumber": 2,
  "customerId": "CUST001",
  "cardType": "VISA",
  "cardNumber": "4111111111111111",
  "expiryDate": "12/28",
  "holderName": "Nguyen Van A",
  "email": "a@example.com"
}
```

## Business Validation

The onboarding worker validates:

- `customerId` is required
- `cardType` is required and must be `VISA`, `MASTERCARD`, or `AMEX`
- `cardNumber` is required and numeric
- `expiryDate` is required and must follow `MM/YY`
- `holderName` is required
- `email` is required and must be a valid email address

Business validation failures are logged and treated as handled. The worker does not call the onboard service and does not retry the message.

## Retry and DLQ Behavior

Retry is handled by SQS.

Both queues use:

- visibility timeout: `60 seconds`
- message retention: `4 days`
- DLQ max receive count: `3`
- SQS managed encryption
- SSL enforcement

Behavior summary:

- preprocessor technical failure: return error, retry, then DLQ
- worker message parse failure: return error, retry, then DLQ
- worker business validation failure: return success, no retry
- onboard-service `2xx`: return success
- onboard-service `4xx`: return success, no retry
- onboard-service `5xx`, timeout, or network error: return error, retry, then DLQ

## Configuration

### `card-onboarding-file-preprocessor`

| Variable | Required | Description |
| --- | --- | --- |
| `OUTPUT_BUCKET_NAME` | Yes | S3 bucket where preprocessing result CSV files are written. |
| `MAX_FILE_SIZE_BYTES` | Yes | Maximum allowed input file size in bytes. |
| `WORKER_QUEUE_URL` | Yes | SQS queue URL for accepted onboarding records. |

### `card-onboarding-worker`

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `ONBOARD_SERVICE_BASE_URL` | Yes | none | Base URL for the onboard service API. |
| `ONBOARD_SERVICE_TIMEOUT` | No | `5s` | HTTP timeout parsed with Go duration syntax. |

## Local Development

Prerequisites:

- Go
- PowerShell for Lambda packaging on Windows
- AWS CDK CLI for infrastructure synthesis/deployment
- AWS credentials configured for CDK operations
- sibling `card-onboarding-services` repository, because `go.mod` contains:

```text
replace github.com/NguyenHoangNguyen22120236/card-onboarding-services => ../card-onboarding-services
```

Common commands:

```sh
make install
make build
make test
```

Package both Lambda functions into `dist/`:

```sh
make lambda-package
```

The packaging script builds Linux AMD64 custom runtime binaries named `bootstrap` and creates:

```text
dist/card-onboarding-file-preprocessor.zip
dist/card-onboarding-worker.zip
```

## Infrastructure

The CDK stack in `infra/` creates:

- input S3 bucket: `card-onboarding-input-bucket-{env}`
- output S3 bucket: `card-onboarding-output-bucket-{env}`
- preprocessor queue and DLQ
- worker queue and DLQ
- both Lambda functions
- S3 event notification to the preprocessor queue
- SQS event sources for both Lambdas
- IAM grants
- CloudWatch log groups with one-month retention

Synthesize the stack from the repository root:

```sh
make cdk-synth ENV=dev ONBOARD_SERVICE_BASE_URL=https://example.internal
```

Optional context values:

| Context | Default | Description |
| --- | --- | --- |
| `env` | `dev` | Environment suffix for resource names. |
| `maxFileSizeBytes` | `10485760` | Preprocessor file size limit. |
| `onboardServiceBaseUrl` | `http://localhost:8080` | Onboard service base URL. |
| `onboardServiceTimeout` | `5s` | Worker onboard-service timeout. |

To deploy directly with CDK, package first, then run CDK from `infra/`:

```sh
make lambda-package
cd infra
cdk deploy -c env=dev -c onboardServiceBaseUrl=https://example.internal
```

## Additional Documentation

- `docs/architecture.md`
- `docs/processing-flow.md`
- `docs/retry-dlq-rules.md`
- `infra/README.md`
