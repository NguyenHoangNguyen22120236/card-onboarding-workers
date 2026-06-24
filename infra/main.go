package main

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudwatch"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambdaeventsources"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3notifications"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssqs"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

const (
	preprocessorName = "card-onboarding-file-preprocessor"
	workerName       = "card-onboarding-worker"
)

type CardOnboardingWorkersStackProps struct {
	awscdk.StackProps
	EnvName                 string
	MaxFileSizeBytes        string
	OnboardServiceBaseURL   string
	OnboardServiceTimeout   string
	PreprocessorAssetZipAbs string
	WorkerAssetZipAbs       string
}

type CardOnboardingMonitoringStackProps struct {
	awscdk.StackProps
	EnvName string
}

func NewCardOnboardingWorkersStack(scope constructs.Construct, id string, props *CardOnboardingWorkersStackProps) awscdk.Stack {
	stack := awscdk.NewStack(scope, jsii.String(id), &props.StackProps)

	inputBucket := newBucket(stack, "InputBucket", fmt.Sprintf("card-onboarding-input-bucket-%s", props.EnvName))
	outputBucket := newBucket(stack, "OutputBucket", fmt.Sprintf("card-onboarding-output-bucket-%s", props.EnvName))

	preprocessorDLQ := newDLQ(stack, "PreprocessorDLQ", fmt.Sprintf("%s-dlq-%s", preprocessorName, props.EnvName))
	preprocessorQueue := newQueue(stack, "PreprocessorQueue", fmt.Sprintf("%s-sqs-%s", preprocessorName, props.EnvName), preprocessorDLQ)

	workerDLQ := newDLQ(stack, "WorkerDLQ", fmt.Sprintf("%s-dlq-%s", workerName, props.EnvName))
	workerQueue := newQueue(stack, "WorkerQueue", fmt.Sprintf("%s-sqs-%s", workerName, props.EnvName), workerDLQ)

	preprocessorFunctionName := fmt.Sprintf("%s-%s", preprocessorName, props.EnvName)
	workerFunctionName := fmt.Sprintf("%s-%s", workerName, props.EnvName)

	preprocessorFunction := newGoLambda(stack, "PreprocessorFunction", preprocessorFunctionName, props.PreprocessorAssetZipAbs, map[string]*string{
		"ENVIRONMENT_NAME":    jsii.String(props.EnvName),
		"OUTPUT_BUCKET_NAME":  outputBucket.BucketName(),
		"MAX_FILE_SIZE_BYTES": jsii.String(props.MaxFileSizeBytes),
		"WORKER_QUEUE_URL":    workerQueue.QueueUrl(),
	})

	workerFunction := newGoLambda(stack, "WorkerFunction", workerFunctionName, props.WorkerAssetZipAbs, map[string]*string{
		"ENVIRONMENT_NAME":         jsii.String(props.EnvName),
		"ONBOARD_SERVICE_BASE_URL": jsii.String(props.OnboardServiceBaseURL),
		"ONBOARD_SERVICE_TIMEOUT":  jsii.String(props.OnboardServiceTimeout),
	})

	inputBucket.AddEventNotification(awss3.EventType_OBJECT_CREATED, awss3notifications.NewSqsDestination(preprocessorQueue))

	preprocessorFunction.AddEventSource(awslambdaeventsources.NewSqsEventSource(preprocessorQueue, &awslambdaeventsources.SqsEventSourceProps{
		BatchSize: jsii.Number(10),
		Enabled:   jsii.Bool(true),
	}))
	workerFunction.AddEventSource(awslambdaeventsources.NewSqsEventSource(workerQueue, &awslambdaeventsources.SqsEventSourceProps{
		BatchSize: jsii.Number(10),
		Enabled:   jsii.Bool(true),
	}))

	inputBucket.GrantRead(preprocessorFunction, nil)
	outputBucket.GrantWrite(preprocessorFunction, nil, nil)
	preprocessorQueue.GrantConsumeMessages(preprocessorFunction)
	workerQueue.GrantSendMessages(preprocessorFunction)
	workerQueue.GrantConsumeMessages(workerFunction)

	awscdk.NewCfnOutput(stack, jsii.String("InputBucketName"), &awscdk.CfnOutputProps{Value: inputBucket.BucketName()})
	awscdk.NewCfnOutput(stack, jsii.String("OutputBucketName"), &awscdk.CfnOutputProps{Value: outputBucket.BucketName()})
	awscdk.NewCfnOutput(stack, jsii.String("PreprocessorQueueUrl"), &awscdk.CfnOutputProps{Value: preprocessorQueue.QueueUrl()})
	awscdk.NewCfnOutput(stack, jsii.String("WorkerQueueUrl"), &awscdk.CfnOutputProps{Value: workerQueue.QueueUrl()})

	return stack
}

func NewCardOnboardingMonitoringStack(scope constructs.Construct, id string, props *CardOnboardingMonitoringStackProps) awscdk.Stack {
	stack := awscdk.NewStack(scope, jsii.String(id), &props.StackProps)

	preprocessorQueueName := fmt.Sprintf("%s-sqs-%s", preprocessorName, props.EnvName)
	preprocessorDLQName := fmt.Sprintf("%s-dlq-%s", preprocessorName, props.EnvName)
	workerQueueName := fmt.Sprintf("%s-sqs-%s", workerName, props.EnvName)
	workerDLQName := fmt.Sprintf("%s-dlq-%s", workerName, props.EnvName)
	preprocessorFunctionName := fmt.Sprintf("%s-%s", preprocessorName, props.EnvName)
	workerFunctionName := fmt.Sprintf("%s-%s", workerName, props.EnvName)

	dashboard := awscloudwatch.NewDashboard(stack, jsii.String("CardOnboardingDashboard"), &awscloudwatch.DashboardProps{
		DashboardName: jsii.String(fmt.Sprintf("card-onboarding-%s-dashboard", props.EnvName)),
	})

	preprocessorFileMetrics := []awscloudwatch.IMetric{
		cardOnboardingMetric(props.EnvName, preprocessorName, "card_onboarding_file_preprocessor.file_received.count", "File received"),
		cardOnboardingMetric(props.EnvName, preprocessorName, "card_onboarding_file_preprocessor.file_rejected.count", "File rejected"),
	}
	preprocessorRecordMetrics := []awscloudwatch.IMetric{
		cardOnboardingMetric(props.EnvName, preprocessorName, "card_onboarding_file_preprocessor.records_accepted.count", "Records accepted"),
		cardOnboardingMetric(props.EnvName, preprocessorName, "card_onboarding_file_preprocessor.records_rejected.count", "Records rejected"),
	}
	workerValidationMetrics := []awscloudwatch.IMetric{
		cardOnboardingMetric(props.EnvName, workerName, "card_onboarding_worker.message_received.count", "Messages received"),
		cardOnboardingMetric(props.EnvName, workerName, "card_onboarding_worker.business_validation_success.count", "Business validation success"),
		cardOnboardingMetric(props.EnvName, workerName, "card_onboarding_worker.business_validation_failed.count", "Business validation failure"),
	}
	workerOnboardMetrics := []awscloudwatch.IMetric{
		cardOnboardingMetric(props.EnvName, workerName, "card_onboarding_worker.onboard_success.count", "Onboard success"),
		cardOnboardingMetric(props.EnvName, workerName, "card_onboarding_worker.onboard_failed.count", "Onboard failure"),
	}
	onboardServiceMetrics := []awscloudwatch.IMetric{
		cardOnboardingMetric(props.EnvName, "onboard-service", "onboard_service.request.count", "Request count"),
		cardOnboardingMetric(props.EnvName, "onboard-service", "onboard_service.failed.count", "Failed count"),
		cardOnboardingP95Metric(props.EnvName, "onboard-service", "onboard_service.response_time_ms", "Response time p95"),
	}
	queueDepthMetrics := []awscloudwatch.IMetric{
		sqsMetric(preprocessorQueueName, "ApproximateNumberOfMessagesVisible", "Preprocessor queue depth", awscloudwatch.Stats_AVERAGE()),
		sqsMetric(workerQueueName, "ApproximateNumberOfMessagesVisible", "Worker queue depth", awscloudwatch.Stats_AVERAGE()),
	}
	dlqDepthMetrics := []awscloudwatch.IMetric{
		sqsMetric(preprocessorDLQName, "ApproximateNumberOfMessagesVisible", "Preprocessor DLQ depth", awscloudwatch.Stats_AVERAGE()),
		sqsMetric(workerDLQName, "ApproximateNumberOfMessagesVisible", "Worker DLQ depth", awscloudwatch.Stats_AVERAGE()),
	}

	dashboard.AddWidgets(
		graphWidget("Files", preprocessorFileMetrics),
		graphWidget("Preprocessor Records", preprocessorRecordMetrics),
	)
	dashboard.AddWidgets(
		graphWidget("Worker Validation", workerValidationMetrics),
		graphWidget("Worker Onboarding", workerOnboardMetrics),
	)
	dashboard.AddWidgets(
		graphWidget("Onboard Service", onboardServiceMetrics),
		graphWidget("Queue Depth", queueDepthMetrics),
	)
	dashboard.AddWidgets(
		graphWidget("DLQ Depth", dlqDepthMetrics),
	)

	newAlarm(stack, "PreprocessorDLQMessagesVisibleAlarm", fmt.Sprintf("card-onboarding-%s-preprocessor-dlq-depth", props.EnvName), sqsMetric(preprocessorDLQName, "ApproximateNumberOfMessagesVisible", "Preprocessor DLQ visible messages", awscloudwatch.Stats_MAXIMUM()), 0)
	newAlarm(stack, "WorkerDLQMessagesVisibleAlarm", fmt.Sprintf("card-onboarding-%s-worker-dlq-depth", props.EnvName), sqsMetric(workerDLQName, "ApproximateNumberOfMessagesVisible", "Worker DLQ visible messages", awscloudwatch.Stats_MAXIMUM()), 0)
	newAlarm(stack, "PreprocessorQueueOldestMessageAgeAlarm", fmt.Sprintf("card-onboarding-%s-preprocessor-queue-oldest-message-age", props.EnvName), sqsMetric(preprocessorQueueName, "ApproximateAgeOfOldestMessage", "Preprocessor queue oldest message age", awscloudwatch.Stats_MAXIMUM()), 300)
	newAlarm(stack, "WorkerQueueOldestMessageAgeAlarm", fmt.Sprintf("card-onboarding-%s-worker-queue-oldest-message-age", props.EnvName), sqsMetric(workerQueueName, "ApproximateAgeOfOldestMessage", "Worker queue oldest message age", awscloudwatch.Stats_MAXIMUM()), 300)
	newAlarm(stack, "PreprocessorLambdaErrorsAlarm", fmt.Sprintf("card-onboarding-%s-preprocessor-lambda-errors", props.EnvName), lambdaMetric(preprocessorFunctionName, "Errors", "Preprocessor Lambda errors"), 0)
	newAlarm(stack, "WorkerLambdaErrorsAlarm", fmt.Sprintf("card-onboarding-%s-worker-lambda-errors", props.EnvName), lambdaMetric(workerFunctionName, "Errors", "Worker Lambda errors"), 0)
	newAlarm(stack, "OnboardServiceFailedAlarm", fmt.Sprintf("card-onboarding-%s-onboard-service-failed", props.EnvName), cardOnboardingMetric(props.EnvName, "onboard-service", "onboard_service.failed.count", "Onboard service failed"), 0)
	newAlarm(stack, "OnboardServiceDBWriteFailedAlarm", fmt.Sprintf("card-onboarding-%s-onboard-service-db-write-failed", props.EnvName), cardOnboardingMetric(props.EnvName, "onboard-service", "onboard_service.db_write_failed.count", "Onboard service DB write failed"), 0)
	newAlarm(stack, "OnboardServiceP95ResponseTimeAlarm", fmt.Sprintf("card-onboarding-%s-onboard-service-p95-response-time", props.EnvName), cardOnboardingP95Metric(props.EnvName, "onboard-service", "onboard_service.response_time_ms", "Onboard service response time p95"), 1000)

	return stack
}

func newBucket(scope constructs.Construct, id string, bucketName string) awss3.Bucket {
	return awss3.NewBucket(scope, jsii.String(id), &awss3.BucketProps{
		BucketName:        jsii.String(bucketName),
		BlockPublicAccess: awss3.BlockPublicAccess_BLOCK_ALL(),
		Encryption:        awss3.BucketEncryption_S3_MANAGED,
		EnforceSSL:        jsii.Bool(true),
		RemovalPolicy:     awscdk.RemovalPolicy_RETAIN,
	})
}

func newDLQ(scope constructs.Construct, id string, queueName string) awssqs.Queue {
	return awssqs.NewQueue(scope, jsii.String(id), &awssqs.QueueProps{
		QueueName:         jsii.String(queueName),
		RetentionPeriod:   awscdk.Duration_Days(jsii.Number(4)),
		VisibilityTimeout: awscdk.Duration_Seconds(jsii.Number(60)),
		Encryption:        awssqs.QueueEncryption_SQS_MANAGED,
		EnforceSSL:        jsii.Bool(true),
	})
}

func newQueue(scope constructs.Construct, id string, queueName string, dlq awssqs.IQueue) awssqs.Queue {
	return awssqs.NewQueue(scope, jsii.String(id), &awssqs.QueueProps{
		QueueName:         jsii.String(queueName),
		RetentionPeriod:   awscdk.Duration_Days(jsii.Number(4)),
		VisibilityTimeout: awscdk.Duration_Seconds(jsii.Number(60)),
		Encryption:        awssqs.QueueEncryption_SQS_MANAGED,
		EnforceSSL:        jsii.Bool(true),
		DeadLetterQueue: &awssqs.DeadLetterQueue{
			MaxReceiveCount: jsii.Number(3),
			Queue:           dlq,
		},
	})
}

func newGoLambda(scope constructs.Construct, id string, functionName string, assetZipAbs string, environment map[string]*string) awslambda.Function {
	logGroup := awslogs.NewLogGroup(scope, jsii.String(id+"LogGroup"), &awslogs.LogGroupProps{
		LogGroupName:  jsii.String(fmt.Sprintf("/aws/lambda/%s", functionName)),
		Retention:     awslogs.RetentionDays_ONE_MONTH,
		RemovalPolicy: awscdk.RemovalPolicy_RETAIN,
	})

	return awslambda.NewFunction(scope, jsii.String(id), &awslambda.FunctionProps{
		FunctionName: jsii.String(functionName),
		Runtime:      awslambda.Runtime_PROVIDED_AL2023(),
		Architecture: awslambda.Architecture_X86_64(),
		Handler:      jsii.String("bootstrap"),
		Code:         awslambda.Code_FromAsset(jsii.String(assetZipAbs), nil),
		Environment:  &environment,
		LogGroup:     logGroup,
		MemorySize:   jsii.Number(256),
		Timeout:      awscdk.Duration_Seconds(jsii.Number(60)),
	})
}

func cardOnboardingMetric(envName string, component string, metricName string, label string) awscloudwatch.Metric {
	return awscloudwatch.NewMetric(&awscloudwatch.MetricProps{
		Namespace:  jsii.String("CardOnboarding"),
		MetricName: jsii.String(metricName),
		DimensionsMap: &map[string]*string{
			"Environment": jsii.String(envName),
			"Component":   jsii.String(component),
		},
		Statistic: awscloudwatch.Stats_SUM(),
		Period:    awscdk.Duration_Minutes(jsii.Number(5)),
		Label:     jsii.String(label),
	})
}

func cardOnboardingP95Metric(envName string, component string, metricName string, label string) awscloudwatch.Metric {
	return awscloudwatch.NewMetric(&awscloudwatch.MetricProps{
		Namespace:  jsii.String("CardOnboarding"),
		MetricName: jsii.String(metricName),
		DimensionsMap: &map[string]*string{
			"Environment": jsii.String(envName),
			"Component":   jsii.String(component),
		},
		Statistic: awscloudwatch.Stats_P(jsii.Number(95)),
		Period:    awscdk.Duration_Minutes(jsii.Number(5)),
		Label:     jsii.String(label),
	})
}

func sqsMetric(queueName string, metricName string, label string, statistic *string) awscloudwatch.Metric {
	return awscloudwatch.NewMetric(&awscloudwatch.MetricProps{
		Namespace:  jsii.String("AWS/SQS"),
		MetricName: jsii.String(metricName),
		DimensionsMap: &map[string]*string{
			"QueueName": jsii.String(queueName),
		},
		Statistic: statistic,
		Period:    awscdk.Duration_Minutes(jsii.Number(5)),
		Label:     jsii.String(label),
	})
}

func lambdaMetric(functionName string, metricName string, label string) awscloudwatch.Metric {
	return awscloudwatch.NewMetric(&awscloudwatch.MetricProps{
		Namespace:  jsii.String("AWS/Lambda"),
		MetricName: jsii.String(metricName),
		DimensionsMap: &map[string]*string{
			"FunctionName": jsii.String(functionName),
		},
		Statistic: awscloudwatch.Stats_SUM(),
		Period:    awscdk.Duration_Minutes(jsii.Number(5)),
		Label:     jsii.String(label),
	})
}

func graphWidget(title string, metrics []awscloudwatch.IMetric) awscloudwatch.GraphWidget {
	return awscloudwatch.NewGraphWidget(&awscloudwatch.GraphWidgetProps{
		Title:  jsii.String(title),
		Left:   &metrics,
		Width:  jsii.Number(12),
		Height: jsii.Number(6),
	})
}

func newAlarm(scope constructs.Construct, id string, alarmName string, metric awscloudwatch.IMetric, threshold float64) awscloudwatch.Alarm {
	return awscloudwatch.NewAlarm(scope, jsii.String(id), &awscloudwatch.AlarmProps{
		AlarmName:          jsii.String(alarmName),
		Metric:             metric,
		Threshold:          jsii.Number(threshold),
		EvaluationPeriods:  jsii.Number(1),
		DatapointsToAlarm:  jsii.Number(1),
		ComparisonOperator: awscloudwatch.ComparisonOperator_GREATER_THAN_THRESHOLD,
		TreatMissingData:   awscloudwatch.TreatMissingData_NOT_BREACHING,
	})
}

func main() {
	app := awscdk.NewApp(nil)
	repoRoot := repositoryRoot()

	envName := contextString(app, "env", "dev")
	stackGroup := contextString(app, "stackGroup", "all")

	switch stackGroup {
	case "all", "workers", "monitoring":
	default:
		panic(fmt.Sprintf("invalid stackGroup %q: expected all, workers, or monitoring", stackGroup))
	}

	if stackGroup == "all" || stackGroup == "workers" {
		NewCardOnboardingWorkersStack(app, "CardOnboardingWorkersStack", &CardOnboardingWorkersStackProps{
			EnvName:                 envName,
			MaxFileSizeBytes:        contextString(app, "maxFileSizeBytes", "10485760"),
			OnboardServiceBaseURL:   contextString(app, "onboardServiceBaseUrl", "http://localhost:8080"),
			OnboardServiceTimeout:   contextString(app, "onboardServiceTimeout", "5s"),
			PreprocessorAssetZipAbs: filepath.Join(repoRoot, "dist", "card-onboarding-file-preprocessor.zip"),
			WorkerAssetZipAbs:       filepath.Join(repoRoot, "dist", "card-onboarding-worker.zip"),
		})
	}
	if stackGroup == "all" || stackGroup == "monitoring" {
		NewCardOnboardingMonitoringStack(app, "CardOnboardingMonitoringStack", &CardOnboardingMonitoringStackProps{
			EnvName: envName,
		})
	}

	app.Synth(nil)
}

func repositoryRoot() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("resolve current file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), ".."))
}

func contextString(app awscdk.App, key string, defaultValue string) string {
	value := app.Node().TryGetContext(jsii.String(key))
	if value == nil {
		return defaultValue
	}
	if stringValue, ok := value.(string); ok && stringValue != "" {
		return stringValue
	}
	return fmt.Sprintf("%v", value)
}
