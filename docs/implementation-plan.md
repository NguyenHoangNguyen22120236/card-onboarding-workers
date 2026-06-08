# Workers Implementation Plan

## Phase 1: Repository Setup

Create repository:

```
card-onboarding-workers
```

Create worker folders:

```
card-onboarding-file-preprocessor
card-onboarding-worker
```

Add root files:

```
go.mod
go.sum
Makefile
pipeline.yaml
deploy.sh
README.md
VERSION
CHANGELOG.md
```

## Phase 2: Implement card-onboarding-file-preprocessor Core Logic

Build local logic first before AWS integration.

Implement:

- S3 event parser
- CSV parser
- CSV structure validator
- CSV mapper
- Preprocessing result generator
- Accepted record mapper

## Phase 3: File Validation

Implement validation rules:

- file extension must be .csv
- file size must be greater than 0
- file size must not exceed configured max file size
- file must be readable
- file must contain header row
- file must contain at least one data row
- CSV header must match expected header
- each data row must have exactly 6 columns
- malformed rows must be rejected

## Phase 4: Preprocessing Output

Generate output CSV:

```
record_id,row_number,customer_id,preprocess_status,error_message
```

Upload result to:

```
s3://card-onboarding-output-bucket-{env}/processed/{jobId}/{sourceFileName}_preprocess_result.csv
```

## Phase 5: Publish Accepted Records to SQS

Publish one message per structurally accepted record to:

```
card-onboarding-worker-sqs-{env}
```

## Phase 6: Implement card-onboarding-worker Core Logic

Implement:

- SQS message parser
- Business validator
- Card number masking
- Onboard service client wrapper
- Response handling

## Phase 7: Business Validation

Implement rules:

- customerId is required
- cardType is required
- cardType must be VISA, MASTERCARD, or AMEX
- cardNumber is required
- cardNumber must be numeric
- expiryDate is required
- expiryDate must be MM/YY
- holderName is required
- email is required
- email must be valid email format

## Phase 8: Retry Behavior

Implement response behavior:

- Business validation failure → return success
- onboard-service 2xx → return success
- onboard-service 4xx → return success
- onboard-service 5xx → return error
- onboard-service timeout → return error

## Phase 9: AWS CDK Infrastructure

Use AWS CDK v2 for Go.

Create:

- S3 input bucket
- S3 output bucket
- SQS queues
- DLQs
- Lambda functions
- IAM roles
- CloudWatch log groups
- Environment variables

## Phase 10: Add Tests

Required tests:

- S3 event parser
- file extension validation
- file size validation
- empty file validation
- CSV header validation
- CSV row column count validation
- preprocessing result generation
- SQS message mapping
- S3 download failure
- SQS publish failure
- SQS message parser
- business validation success
- missing customerId
- invalid cardType
- invalid cardNumber
- invalid expiryDate
- invalid email
- onboard-service 2xx handling
- onboard-service 4xx no retry
- onboard-service 5xx retry
- timeout retry

## Phase 11: Smoke Tests

Smoke tests should cover:

- Upload valid CSV to S3
- Verify preprocessing output file is created
- Verify accepted records are published to worker SQS
- Verify business validation
- Verify onboard-service call
- Verify invalid records are not sent to onboard-service
- Verify failed technical messages go to DLQ after retries

## Phase 12: Add README and CHANGELOG

Each worker must have:

- README.md
- CHANGELOG.md
