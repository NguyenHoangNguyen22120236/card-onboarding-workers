package observability

import (
	"encoding/json"
	"log"
	"time"
)

const (
	Namespace = "CardOnboarding"

	UnitCount        = "Count"
	UnitMilliseconds = "Milliseconds"
)

type Metric struct {
	Name  string
	Value float64
	Unit  string
}

type Fields struct {
	Environment   string
	Component     string
	CorrelationID string
	JobID         string
	RecordID      string
	CustomerID    string
	SourceFile    string
	RowNumber     int
	Step          string
	Status        string
	DurationMs    int64
	ErrorCode     string
	ErrorMessage  string
}

type awsMetadata struct {
	Timestamp         int64                    `json:"Timestamp"`
	CloudWatchMetrics []cloudWatchMetricConfig `json:"CloudWatchMetrics"`
}

type cloudWatchMetricConfig struct {
	Namespace  string             `json:"Namespace"`
	Dimensions [][]string         `json:"Dimensions"`
	Metrics    []cloudWatchMetric `json:"Metrics"`
}

type cloudWatchMetric struct {
	Name string `json:"Name"`
	Unit string `json:"Unit,omitempty"`
}

func LogMetric(metric Metric, fields Fields) {
	message, err := BuildMetric(metric, fields, time.Now())
	if err != nil {
		log.Println(err.Error())
		return
	}

	log.Println(message)
}

func BuildMetric(metric Metric, fields Fields, timestamp time.Time) (string, error) {
	event := map[string]any{
		"_aws": awsMetadata{
			Timestamp: timestamp.UnixMilli(),
			CloudWatchMetrics: []cloudWatchMetricConfig{
				{
					Namespace:  Namespace,
					Dimensions: [][]string{{"Environment", "Component"}},
					Metrics: []cloudWatchMetric{
						{
							Name: metric.Name,
							Unit: metric.Unit,
						},
					},
				},
			},
		},
		"Environment": fields.Environment,
		"Component":   fields.Component,
		metric.Name:   metric.Value,
	}

	addString(event, "correlationId", fields.CorrelationID)
	addString(event, "jobId", fields.JobID)
	addString(event, "recordId", fields.RecordID)
	addString(event, "customerId", fields.CustomerID)
	addString(event, "sourceFile", fields.SourceFile)
	addInt(event, "rowNumber", fields.RowNumber)
	addString(event, "step", fields.Step)
	addString(event, "status", fields.Status)
	addInt64(event, "durationMs", fields.DurationMs)
	addString(event, "errorCode", fields.ErrorCode)
	addString(event, "errorMessage", fields.ErrorMessage)

	payload, err := json.Marshal(event)
	if err != nil {
		return "", err
	}

	return string(payload), nil
}

func addString(event map[string]any, key string, value string) {
	if value != "" {
		event[key] = value
	}
}

func addInt(event map[string]any, key string, value int) {
	if value != 0 {
		event[key] = value
	}
}

func addInt64(event map[string]any, key string, value int64) {
	if value != 0 {
		event[key] = value
	}
}
