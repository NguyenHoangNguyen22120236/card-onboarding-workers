package observability

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBuildMetricUsesOnlyEnvironmentAndComponentDimensions(t *testing.T) {
	message, err := BuildMetric(
		Metric{
			Name:  "RecordsProcessed",
			Value: 1,
			Unit:  UnitCount,
		},
		Fields{
			Environment:   "dev",
			Component:     "card-onboarding-worker",
			CorrelationID: "corr-123",
			JobID:         "job-123",
			RecordID:      "record-123",
			CustomerID:    "customer-123",
			SourceFile:    "cards.csv",
			RowNumber:     2,
			Step:          "onboard-card",
			Status:        "success",
			DurationMs:    25,
		},
		time.UnixMilli(1000),
	)
	if err != nil {
		t.Fatalf("BuildMetric returned error: %v", err)
	}

	var event map[string]any
	if err := json.Unmarshal([]byte(message), &event); err != nil {
		t.Fatalf("metric log is not valid JSON: %v", err)
	}

	if event["correlationId"] != "corr-123" {
		t.Fatalf("correlationId = %v, want %q", event["correlationId"], "corr-123")
	}
	if event["RecordsProcessed"] != float64(1) {
		t.Fatalf("RecordsProcessed = %v, want %v", event["RecordsProcessed"], float64(1))
	}

	awsMetadata := event["_aws"].(map[string]any)
	metricConfigs := awsMetadata["CloudWatchMetrics"].([]any)
	metricConfig := metricConfigs[0].(map[string]any)
	dimensions := metricConfig["Dimensions"].([]any)
	dimensionSet := dimensions[0].([]any)

	if len(dimensionSet) != 2 {
		t.Fatalf("dimension count = %d, want 2", len(dimensionSet))
	}
	if dimensionSet[0] != "Environment" || dimensionSet[1] != "Component" {
		t.Fatalf("dimensions = %v, want [Environment Component]", dimensionSet)
	}
}
