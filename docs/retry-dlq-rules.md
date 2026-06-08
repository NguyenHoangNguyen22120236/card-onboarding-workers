# Retry and DLQ Rules

## 1. Overview

Retry is handled by SQS and DLQ.

The system must not store retry count in DynamoDB.

## 2. Queue Naming Standard

SQS queue naming:

```
{service-name}-sqs-{env}
```

DLQ naming:

```
{service-name}-dlq-{env}
```

Required queues:

```
card-onboarding-file-preprocessor-sqs-{env}
card-onboarding-file-preprocessor-dlq-{env}
card-onboarding-worker-sqs-{env}
card-onboarding-worker-dlq-{env}
```

## 3. card-onboarding-file-preprocessor Queue

Queue:

```
card-onboarding-file-preprocessor-sqs-{env}
```

DLQ:

```
card-onboarding-file-preprocessor-dlq-{env}
```

Configuration:

```
maxReceiveCount: 3
visibilityTimeout: 60 seconds
messageRetentionPeriod: 4 days
```

## 4. File Preprocessor Retry Rules

The file preprocessor should return an error when it cannot complete technical processing.

Examples of technical failures:

- Cannot parse SQS event
- Cannot extract S3 bucket or object key
- Cannot download CSV file from S3
- Cannot validate required file-level checks
- Cannot upload preprocessing result file to S3
- Cannot publish accepted records to worker SQS

Behavior:

- Return error from Lambda
- Do not delete message
- SQS retries the message
- After 3 failed receives, move message to DLQ

## 5. card-onboarding-worker Queue

Queue:

```
card-onboarding-worker-sqs-{env}
```

DLQ:

```
card-onboarding-worker-dlq-{env}
```

Configuration:

```
maxReceiveCount: 3
visibilityTimeout: 60 seconds
messageRetentionPeriod: 4 days
```

## 6. Business Validation Failure

If business validation fails:

- Log validation failure
- Emit failure metric
- Return success
- Message is not retried
- Message does not go to DLQ
- Do not call onboard-service

Reason:

Business-invalid data will not become valid by retrying.

## 7. onboard-service 2xx Response

If onboard-service returns 2xx:

- Complete successfully
- Message is deleted by Lambda/SQS integration

## 8. onboard-service 4xx Response

If onboard-service returns 4xx:

- Log business failure
- Emit failure metric
- Return success
- Message is not retried
- Message does not go to DLQ

Reason:

A 4xx response usually means the request is invalid or rejected. Retrying the same message will likely fail again.

## 9. onboard-service 5xx or Timeout

If onboard-service returns 5xx or timeout:

- Return error
- SQS retries the message
- After 3 failed receives, move message to DLQ

Reason:

A 5xx or timeout is a technical failure and may succeed later.
