# Card Onboarding Workers Architecture

## 1. Overview

The `card-onboarding-workers` repository contains the event-driven workers for the Card Onboarding File Processing Platform.

This repository owns two Lambda workers:

1. `card-onboarding-file-preprocessor`
2. `card-onboarding-worker`

The workers are responsible for processing uploaded CSV files, validating records, publishing messages, and calling the onboarding service.

## 2. High-Level Flow

```
Manual CSV Upload
    ↓
S3 Input Bucket
    ↓
S3 ObjectCreated Event
    ↓
card-onboarding-file-preprocessor-sqs-{env}
    ↓
card-onboarding-file-preprocessor
    ↓
S3 Output Bucket
    ↓
card-onboarding-worker-sqs-{env}
    ↓
card-onboarding-worker
    ↓
onboard-service
```

## 3. AWS Components

Required AWS resources:

- S3 input bucket
- S3 output bucket
- card-onboarding-file-preprocessor-sqs-{env}
- card-onboarding-file-preprocessor-dlq-{env}
- card-onboarding-worker-sqs-{env}
- card-onboarding-worker-dlq-{env}
- card-onboarding-file-preprocessor Lambda
- card-onboarding-worker Lambda
- Lambda IAM roles
- CloudWatch log groups
- Environment variables

## 4. card-onboarding-file-preprocessor

The file preprocessor consumes S3 file-created events from SQS.

Responsibilities:

- Parse SQS message
- Extract S3 bucket and object key
- Download CSV file from S3
- Validate file extension
- Validate file size
- Validate file is not empty
- Validate CSV header
- Validate CSV row column count
- Reject malformed rows
- Generate jobId
- Generate recordId for each accepted row
- Create preprocessing result CSV
- Upload preprocessing result CSV to S3 output bucket
- Publish each accepted record to card-onboarding-worker-sqs-{env}
- Emit logs and metrics

## 5. card-onboarding-worker

The onboarding worker consumes accepted record messages from SQS.

Responsibilities:

- Parse onboarding message
- Validate business fields
- Mask card number before logging
- Do not call onboard-service if business validation fails
- Call onboard-service using generated Go client
- Handle 2xx, 4xx, 5xx, and timeout responses
- Let SQS retry technical failures
- Let SQS move failed technical messages to DLQ after max receive count

## 6. Important Rule

The card-onboarding-worker does not download the original CSV from S3.

It only consumes individual accepted record messages from:

```
card-onboarding-worker-sqs-{env}
```

The original CSV is only downloaded by:

```
card-onboarding-file-preprocessor
```

## 7. Observability

Structured logs must include:

- correlationId
- jobId
- recordId
- customerId
- sourceFile
- rowNumber
- component
- step
- status
- durationMs
- errorCode
- errorMessage

Card numbers must be masked in logs.
