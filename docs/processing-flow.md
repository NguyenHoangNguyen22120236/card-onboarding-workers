# Processing Flow

## 1. Overview

This document describes how files and records move through the worker system.

The flow has two main stages:

```
1. File-level and CSV structure validation
2. Business validation and onboarding
```

## 2. Input File

CSV files are manually uploaded to the S3 input bucket.

Example path:

```
s3://card-onboarding-input-bucket-{env}/manual-upload/cards_20260606.csv
```

Required CSV header:

```
customer_id,card_type,card_number,expiry_date,holder_name,email
```

Example CSV:

```
customer_id,card_type,card_number,expiry_date,holder_name,email
CUST001,VISA,4111111111111111,12/28,Nguyen Van A,a@example.com
CUST002,MASTERCARD,5555555555554444,10/27,Tran Van B,b@example.com
```

## 3. File Preprocessor Flow

1. S3 receives CSV file.
2. S3 sends ObjectCreated event to SQS.
3. card-onboarding-file-preprocessor consumes the SQS message.
4. Worker extracts bucket name and object key.
5. Worker downloads CSV file from S3.
6. Worker validates file and CSV structure.
7. Worker generates preprocessing result CSV.
8. Worker uploads result CSV to S3 output bucket.
9. Worker publishes accepted records to card-onboarding-worker-sqs-{env}.

## 4. File and Structure Validation

The file preprocessor validates only file-level and CSV structure rules.

Validation rules:

- file extension must be .csv
- file size must be greater than 0
- file size must not exceed configured max file size
- file must be readable
- file must contain header row
- file must contain at least one data row
- CSV header must match expected header
- each data row must have exactly 6 columns
- malformed rows must be rejected

## 5. Preprocessing Output File

Output path:

```
s3://card-onboarding-output-bucket-{env}/processed/{jobId}/{sourceFileName}_preprocess_result.csv
```

Output CSV format:

```
record_id,row_number,customer_id,preprocess_status,error_message
REC-001,2,CUST001,ACCEPTED,
REC-002,3,,REJECTED,malformed row: expected 6 columns but got 4
```

Preprocess status values:

- ACCEPTED
- REJECTED

## 6. Message Published to Worker SQS

For each structurally accepted record, the file preprocessor publishes a JSON message.

Example:

```json
{
  "correlationId": "corr-123",
  "jobId": "JOB-20260606-001",
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

## 7. Card Onboarding Worker Flow

1. Consume message from card-onboarding-worker-sqs-{env}.
2. Parse onboarding message.
3. Validate business fields.
4. Mask card number before logging.
5. If business validation fails, log error, emit metric, and return success.
6. If business validation passes, call onboard-service using generated client.
7. Return success for onboard-service 2xx.
8. Return success for onboard-service 4xx to avoid retry.
9. Return error for onboard-service 5xx or timeout so SQS can retry.
10. Emit logs and metrics.

## 8. Business Validation Rules

The card-onboarding-worker validates business fields.

Rules:

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

## 9. Business Validation Failure

If business validation fails:

- Do not call onboard-service
- Log validation failure
- Emit card_onboarding_worker.business_validation_failed.count
- Return success from Lambda handler
- Message must not be retried
- Message must not go to DLQ
