package smoketest

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamotypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type smokeApp struct {
	cfg    smokeConfig
	s3     *s3.Client
	sqs    *sqs.Client
	dynamo *dynamodb.Client
	http   *http.Client
	runID  string
}

type onboardingMessage struct {
	CorrelationID string `json:"correlationId"`
	JobID         string `json:"jobId"`
	RecordID      string `json:"recordId"`
	SourceFile    string `json:"sourceFile"`
	RowNumber     int    `json:"rowNumber"`
	CustomerID    string `json:"customerId"`
	CardType      string `json:"cardType"`
	CardNumber    string `json:"cardNumber"`
	ExpiryDate    string `json:"expiryDate"`
	HolderName    string `json:"holderName"`
	Email         string `json:"email"`
}

type onboardingStatus struct {
	CustomerID                 string `json:"customerId"`
	OverallStatus              string `json:"overallStatus"`
	CustomerRegistrationStatus string `json:"customerRegistrationStatus"`
	InterestDetailsStatus      string `json:"interestDetailsStatus"`
	AccountOnboardingStatus    string `json:"accountOnboardingStatus"`
}

type onboardResponse struct {
	CustomerID     string `json:"customerId"`
	CoreCustomerID string `json:"coreCustomerId"`
	AccountID      string `json:"accountId"`
	CardID         string `json:"cardId"`
	Status         string `json:"status"`
}

type accountDetailsItem struct {
	CustomerID       string
	CoreCustomerID   string
	ProductCode      string
	InterestRate     string
	InterestType     string
	Currency         string
	AccountID        string
	CardID           string
	CardType         string
	CardNumberMasked string
}

func TestSmokeFullCardOnboardingPlatform(t *testing.T) {
	ctx := context.Background()
	app := newSmokeApp(t, ctx)

	valid := app.uploadCSV(t, ctx, "cards_valid.csv")
	validRows := app.waitForPreprocessResult(t, ctx, valid.sourceFile)
	assertPreprocessStatus(t, validRows, "CUST_SMOKE_VALID_"+app.runID, "ACCEPTED")

	if app.cfg.RequireQueueObservation {
		msg := app.waitForWorkerQueueMessage(t, ctx, valid.sourceFile, "CUST_SMOKE_VALID_"+app.runID)
		if msg.CardNumber != "4111111111111111" || msg.CardType != "VISA" {
			t.Fatalf("worker queue message = %#v, want accepted valid CSV record", msg)
		}
	}

	validStatus := app.waitForStatus(t, "CUST_SMOKE_VALID_"+app.runID, func(status onboardingStatus) bool {
		return status.OverallStatus == "SUCCEEDED"
	})
	if validStatus.CustomerRegistrationStatus != "SUCCEEDED" ||
		validStatus.InterestDetailsStatus != "SUCCEEDED" ||
		validStatus.AccountOnboardingStatus != "SUCCEEDED" {
		t.Fatalf("valid status = %#v, want all onboarding steps SUCCEEDED", validStatus)
	}

	validRequest := valid.firstMessage(t)
	resp := app.onboardDirect(t, validRequest)
	if resp.CoreCustomerID == "" || resp.AccountID == "" || resp.CardID == "" || resp.Status != "SUCCEEDED" {
		t.Fatalf("repeat onboard response = %#v, want saved account details and SUCCEEDED status", resp)
	}
	app.assertAccountDetailsSaved(t, ctx, "CUST_SMOKE_VALID_"+app.runID)

	structure := app.uploadCSV(t, ctx, "cards_invalid_structure.csv")
	structureRows := app.waitForPreprocessResult(t, ctx, structure.sourceFile)
	assertPreprocessStatus(t, structureRows, "CUST_SMOKE_STRUCTURE_OK_"+app.runID, "ACCEPTED")
	assertPreprocessStatus(t, structureRows, "CUST_SMOKE_STRUCTURE_BAD_"+app.runID, "REJECTED")

	businessInvalid := app.uploadCSV(t, ctx, "cards_invalid_business.csv")
	businessRows := app.waitForPreprocessResult(t, ctx, businessInvalid.sourceFile)
	assertPreprocessStatus(t, businessRows, "CUST_SMOKE_BUSINESS_INVALID_"+app.runID, "ACCEPTED")
	app.assertStatusNotFoundFor(t, "CUST_SMOKE_BUSINESS_INVALID_"+app.runID, 30*time.Second)
	app.assertNoDLQMessageFor(t, ctx, "CUST_SMOKE_BUSINESS_INVALID_"+app.runID, 15*time.Second)

	interestFailure := app.uploadCSV(t, ctx, "cards_resume_interest_failure.csv")
	app.waitForPreprocessResult(t, ctx, interestFailure.sourceFile)
	failedStatus := app.waitForStatus(t, "CUST_FAIL_INTEREST", func(status onboardingStatus) bool {
		return status.CustomerRegistrationStatus == "SUCCEEDED" && status.InterestDetailsStatus == "FAILED"
	})
	if failedStatus.OverallStatus != "FAILED" {
		t.Fatalf("interest failure status = %#v, want overall FAILED", failedStatus)
	}

	beforeRetry := app.getStatus(t, "CUST_FAIL_INTEREST")
	_, retryStatusCode := app.onboardDirectStatus(interestFailure.firstMessage(t))
	if retryStatusCode < http.StatusInternalServerError {
		t.Fatalf("retry same customer status code = %d, want technical failure from account service", retryStatusCode)
	}
	afterRetry := app.getStatus(t, "CUST_FAIL_INTEREST")
	if beforeRetry.CustomerRegistrationStatus != "SUCCEEDED" ||
		afterRetry.CustomerRegistrationStatus != "SUCCEEDED" ||
		afterRetry.InterestDetailsStatus != "FAILED" {
		t.Fatalf("resume status before=%#v after=%#v, want customer step preserved and interest FAILED", beforeRetry, afterRetry)
	}

	dlqUpload := app.uploadCSV(t, ctx, "cards_dlq.csv")
	dlqMessage := dlqUpload.firstMessage(t)
	dlqMessage.SourceFile = "cards_dlq_" + app.runID + ".csv"
	app.sendWorkerMessage(t, ctx, dlqMessage)
	if app.cfg.RequireDLQObservation {
		app.waitForDLQMessage(t, ctx, dlqMessage.SourceFile, dlqMessage.CustomerID)
	}
}

type uploadedCSV struct {
	fixture    string
	sourceFile string
	body       string
}

func newSmokeApp(t *testing.T, ctx context.Context) *smokeApp {
	t.Helper()

	cfg := loadSmokeConfig(t)
	awsConfig, err := awscfg.LoadDefaultConfig(ctx, awscfg.WithRegion(cfg.Region))
	if err != nil {
		t.Fatalf("load AWS config: %v", err)
	}

	return &smokeApp{
		cfg:    cfg,
		s3:     s3.NewFromConfig(awsConfig),
		sqs:    sqs.NewFromConfig(awsConfig),
		dynamo: dynamodb.NewFromConfig(awsConfig),
		http:   &http.Client{Timeout: 10 * time.Second},
		runID:  time.Now().UTC().Format("20060102T150405"),
	}
}

func (a *smokeApp) uploadCSV(t *testing.T, ctx context.Context, fixture string) uploadedCSV {
	t.Helper()

	bodyBytes, err := os.ReadFile(filepath.Join("data", fixture))
	if err != nil {
		t.Fatalf("read fixture %s: %v", fixture, err)
	}
	body := strings.ReplaceAll(string(bodyBytes), "{{RUN_ID}}", a.runID)
	sourceFile := strings.TrimSuffix(fixture, ".csv") + "_" + a.runID + ".csv"
	key := strings.Trim(a.cfg.SourcePrefix+"/"+a.runID+"/"+sourceFile, "/")

	_, err = a.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(a.cfg.InputBucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader([]byte(body)),
		ContentType: aws.String("text/csv"),
	})
	if err != nil {
		t.Fatalf("upload %s to s3://%s/%s: %v", fixture, a.cfg.InputBucket, key, err)
	}

	return uploadedCSV{fixture: fixture, sourceFile: sourceFile, body: body}
}

func (u uploadedCSV) firstMessage(t *testing.T) onboardingMessage {
	t.Helper()

	reader := csv.NewReader(strings.NewReader(u.body))
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("parse uploaded CSV %s: %v", u.fixture, err)
	}
	if len(rows) < 2 {
		t.Fatalf("uploaded CSV %s has no data rows", u.fixture)
	}

	row := rows[1]
	return onboardingMessage{
		CorrelationID: "smoke-" + strings.TrimSuffix(u.sourceFile, ".csv"),
		JobID:         "smoke-" + strings.TrimSuffix(u.sourceFile, ".csv"),
		RecordID:      "REC-SMOKE",
		SourceFile:    u.sourceFile,
		RowNumber:     2,
		CustomerID:    row[0],
		CardType:      row[1],
		CardNumber:    row[2],
		ExpiryDate:    row[3],
		HolderName:    row[4],
		Email:         row[5],
	}
}

func (a *smokeApp) waitForPreprocessResult(t *testing.T, ctx context.Context, sourceFile string) [][]string {
	t.Helper()

	suffix := "/" + sourceFile + "_preprocess_result.csv"
	var resultKey string
	err := eventually(a.cfg.PollTimeout, a.cfg.PollInterval, func() error {
		out, err := a.s3.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket: aws.String(a.cfg.OutputBucket),
			Prefix: aws.String("processed/"),
		})
		if err != nil {
			return err
		}
		for _, object := range out.Contents {
			key := aws.ToString(object.Key)
			if strings.HasSuffix(key, suffix) {
				resultKey = key
				return nil
			}
		}
		return fmt.Errorf("preprocess result with suffix %s not found", suffix)
	})
	if err != nil {
		t.Fatalf("wait for preprocess result %s: %v", sourceFile, err)
	}

	out, err := a.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(a.cfg.OutputBucket),
		Key:    aws.String(resultKey),
	})
	if err != nil {
		t.Fatalf("download preprocess result %s: %v", resultKey, err)
	}
	defer out.Body.Close()

	body, err := io.ReadAll(out.Body)
	if err != nil {
		t.Fatalf("read preprocess result %s: %v", resultKey, err)
	}
	rows, err := csv.NewReader(bytes.NewReader(body)).ReadAll()
	if err != nil {
		t.Fatalf("parse preprocess result %s: %v", resultKey, err)
	}
	return rows
}

func assertPreprocessStatus(t *testing.T, rows [][]string, customerID string, status string) {
	t.Helper()

	for _, row := range rows[1:] {
		if len(row) >= 4 && row[2] == customerID {
			if row[3] != status {
				t.Fatalf("preprocess status for %s = %s, want %s; row=%#v", customerID, row[3], status, row)
			}
			return
		}
	}
	t.Fatalf("preprocess result missing customer %s; rows=%#v", customerID, rows)
}

func (a *smokeApp) waitForWorkerQueueMessage(t *testing.T, ctx context.Context, sourceFile, customerID string) onboardingMessage {
	t.Helper()

	var found onboardingMessage
	err := eventually(a.cfg.PollTimeout, a.cfg.PollInterval, func() error {
		out, err := a.sqs.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(a.cfg.WorkerQueueURL),
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     2,
			VisibilityTimeout:   a.cfg.QueueVisibilityTimeout,
		})
		if err != nil {
			return err
		}
		for _, message := range out.Messages {
			var msg onboardingMessage
			if err := json.Unmarshal([]byte(aws.ToString(message.Body)), &msg); err == nil &&
				msg.SourceFile == sourceFile && msg.CustomerID == customerID {
				found = msg
				a.releaseMessage(ctx, a.cfg.WorkerQueueURL, message)
				return nil
			}
			a.releaseMessage(ctx, a.cfg.WorkerQueueURL, message)
		}
		return fmt.Errorf("worker queue message for %s/%s not observed", sourceFile, customerID)
	})
	if err != nil {
		t.Fatalf("wait for worker queue message: %v", err)
	}
	return found
}

func (a *smokeApp) sendWorkerMessage(t *testing.T, ctx context.Context, msg onboardingMessage) {
	t.Helper()

	body, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal worker message: %v", err)
	}
	_, err = a.sqs.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(a.cfg.WorkerQueueURL),
		MessageBody: aws.String(string(body)),
	})
	if err != nil {
		t.Fatalf("send worker message: %v", err)
	}
}

func (a *smokeApp) waitForDLQMessage(t *testing.T, ctx context.Context, sourceFile, customerID string) {
	t.Helper()

	err := eventually(a.cfg.PollTimeout+2*time.Minute, a.cfg.PollInterval, func() error {
		out, err := a.sqs.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(a.cfg.WorkerDLQURL),
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     2,
			VisibilityTimeout:   a.cfg.QueueVisibilityTimeout,
		})
		if err != nil {
			return err
		}
		for _, message := range out.Messages {
			body := aws.ToString(message.Body)
			if strings.Contains(body, sourceFile) && strings.Contains(body, customerID) {
				a.releaseMessage(ctx, a.cfg.WorkerDLQURL, message)
				return nil
			}
			a.releaseMessage(ctx, a.cfg.WorkerDLQURL, message)
		}
		return fmt.Errorf("DLQ message for %s/%s not observed", sourceFile, customerID)
	})
	if err != nil {
		t.Fatalf("wait for worker DLQ message: %v", err)
	}
}

func (a *smokeApp) assertNoDLQMessageFor(t *testing.T, ctx context.Context, customerID string, duration time.Duration) {
	t.Helper()

	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		out, err := a.sqs.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(a.cfg.WorkerDLQURL),
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     2,
			VisibilityTimeout:   a.cfg.QueueVisibilityTimeout,
		})
		if err != nil {
			t.Fatalf("receive DLQ messages: %v", err)
		}
		for _, message := range out.Messages {
			body := aws.ToString(message.Body)
			a.releaseMessage(ctx, a.cfg.WorkerDLQURL, message)
			if strings.Contains(body, customerID) {
				t.Fatalf("business-invalid customer %s was found in worker DLQ body=%s", customerID, body)
			}
		}
		time.Sleep(a.cfg.PollInterval)
	}
}

func (a *smokeApp) releaseMessage(ctx context.Context, queueURL string, message sqstypes.Message) {
	if message.ReceiptHandle == nil {
		return
	}
	_, _ = a.sqs.ChangeMessageVisibility(ctx, &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws.String(queueURL),
		ReceiptHandle:     message.ReceiptHandle,
		VisibilityTimeout: 0,
	})
}

func (a *smokeApp) waitForStatus(t *testing.T, customerID string, accept func(onboardingStatus) bool) onboardingStatus {
	t.Helper()

	var status onboardingStatus
	err := eventually(a.cfg.PollTimeout, a.cfg.PollInterval, func() error {
		current, code := a.getStatusWithCode(t, customerID)
		if code == http.StatusNotFound {
			return fmt.Errorf("status for %s not found yet", customerID)
		}
		if code != http.StatusOK {
			return fmt.Errorf("status for %s returned HTTP %d", customerID, code)
		}
		if !accept(current) {
			return fmt.Errorf("status for %s = %#v", customerID, current)
		}
		status = current
		return nil
	})
	if err != nil {
		t.Fatalf("wait for status %s: %v", customerID, err)
	}
	return status
}

func (a *smokeApp) assertStatusNotFoundFor(t *testing.T, customerID string, duration time.Duration) {
	t.Helper()

	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		status, code := a.getStatusWithCode(t, customerID)
		if code == http.StatusOK {
			t.Fatalf("status for %s unexpectedly exists: %#v", customerID, status)
		}
		if code != http.StatusNotFound {
			t.Fatalf("status for %s returned HTTP %d, want 404 while business-invalid record is handled", customerID, code)
		}
		time.Sleep(a.cfg.PollInterval)
	}
}

func (a *smokeApp) getStatus(t *testing.T, customerID string) onboardingStatus {
	t.Helper()

	status, code := a.getStatusWithCode(t, customerID)
	if code != http.StatusOK {
		t.Fatalf("get status %s returned HTTP %d", customerID, code)
	}
	return status
}

func (a *smokeApp) getStatusWithCode(t *testing.T, customerID string) (onboardingStatus, int) {
	t.Helper()

	endpoint := fmt.Sprintf("%s/internal/cards/%s/status", a.cfg.OnboardServiceBaseURL, url.PathEscape(customerID))
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		t.Fatalf("build status request: %v", err)
	}
	req.Header.Set("X-Correlation-Id", "smoke-"+a.runID)

	resp, err := a.http.Do(req)
	if err != nil {
		t.Fatalf("get status %s: %v", customerID, err)
	}
	defer resp.Body.Close()

	var status onboardingStatus
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
			t.Fatalf("decode status %s: %v", customerID, err)
		}
	}
	return status, resp.StatusCode
}

func (a *smokeApp) onboardDirect(t *testing.T, msg onboardingMessage) onboardResponse {
	t.Helper()

	resp, statusCode := a.onboardDirectStatus(msg)
	if statusCode != http.StatusOK {
		t.Fatalf("direct onboard %s returned HTTP %d", msg.CustomerID, statusCode)
	}
	return resp
}

func (a *smokeApp) onboardDirectStatus(msg onboardingMessage) (onboardResponse, int) {
	endpoint := a.cfg.OnboardServiceBaseURL + "/internal/cards/onboard"
	body, err := json.Marshal(msg)
	if err != nil {
		return onboardResponse{}, 0
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return onboardResponse{}, 0
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Correlation-Id", msg.CorrelationID)

	resp, err := a.http.Do(req)
	if err != nil {
		return onboardResponse{}, 0
	}
	defer resp.Body.Close()

	var onboardResp onboardResponse
	if resp.StatusCode == http.StatusOK {
		_ = json.NewDecoder(resp.Body).Decode(&onboardResp)
	}
	return onboardResp, resp.StatusCode
}

func (a *smokeApp) assertAccountDetailsSaved(t *testing.T, ctx context.Context, customerID string) {
	t.Helper()

	if a.cfg.AccountDetailsTableName == "" {
		t.Log("SMOKE_ACCOUNT_DETAILS_TABLE not set; account detail persistence verified through repeat onboard response")
		return
	}

	out, err := a.dynamo.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(a.cfg.AccountDetailsTableName),
		Key: map[string]dynamotypes.AttributeValue{
			"customerId": &dynamotypes.AttributeValueMemberS{Value: customerID},
		},
	})
	if err != nil {
		t.Fatalf("get account details item: %v", err)
	}
	if len(out.Item) == 0 {
		t.Fatalf("account details item for %s not found", customerID)
	}

	details := accountDetailsItem{
		CustomerID:       dynamoString(out.Item, "customerId"),
		CoreCustomerID:   dynamoString(out.Item, "coreCustomerId"),
		ProductCode:      dynamoString(out.Item, "productCode"),
		InterestRate:     dynamoNumber(out.Item, "interestRate"),
		InterestType:     dynamoString(out.Item, "interestType"),
		Currency:         dynamoString(out.Item, "currency"),
		AccountID:        dynamoString(out.Item, "accountId"),
		CardID:           dynamoString(out.Item, "cardId"),
		CardType:         dynamoString(out.Item, "cardType"),
		CardNumberMasked: dynamoString(out.Item, "cardNumberMasked"),
	}
	if details.CoreCustomerID == "" ||
		details.ProductCode == "" ||
		details.AccountID == "" ||
		details.CardID == "" ||
		details.CardNumberMasked == "" {
		t.Fatalf("account details item = %#v, want customer, interest, account, and masked card details", details)
	}
}

func dynamoString(item map[string]dynamotypes.AttributeValue, key string) string {
	value, ok := item[key].(*dynamotypes.AttributeValueMemberS)
	if !ok {
		return ""
	}
	return value.Value
}

func dynamoNumber(item map[string]dynamotypes.AttributeValue, key string) string {
	value, ok := item[key].(*dynamotypes.AttributeValueMemberN)
	if !ok {
		return ""
	}
	return value.Value
}

func eventually(timeout, interval time.Duration, check func() error) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		if err := check(); err != nil {
			lastErr = err
			time.Sleep(interval)
			continue
		}
		return nil
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("condition not met before %s", timeout)
}
