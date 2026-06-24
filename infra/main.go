package main

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/aws/aws-cdk-go/awscdk/v2"
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

func main() {
	app := awscdk.NewApp(nil)
	repoRoot := repositoryRoot()

	envName := contextString(app, "env", "dev")
	NewCardOnboardingWorkersStack(app, "CardOnboardingWorkersStack", &CardOnboardingWorkersStackProps{
		EnvName:                 envName,
		MaxFileSizeBytes:        contextString(app, "maxFileSizeBytes", "10485760"),
		OnboardServiceBaseURL:   contextString(app, "onboardServiceBaseUrl", "http://localhost:8080"),
		OnboardServiceTimeout:   contextString(app, "onboardServiceTimeout", "5s"),
		PreprocessorAssetZipAbs: filepath.Join(repoRoot, "dist", "card-onboarding-file-preprocessor.zip"),
		WorkerAssetZipAbs:       filepath.Join(repoRoot, "dist", "card-onboarding-worker.zip"),
	})

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
