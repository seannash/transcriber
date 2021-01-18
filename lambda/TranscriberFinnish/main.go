package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"example.com/transcribe/internal/transcribe"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/sqs"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func HandleRequest(ctx context.Context, event events.CloudWatchEvent) error {
	mapStructure := &map[string]string{}
	_ = json.Unmarshal(event.Detail, mapStructure)
	jobName := (*mapStructure)["TranscriptionJobName"]
	status := (*mapStructure)["TranscriptionJobStatus"]
	err := transcribe.SetDatabaseRecordStatus(databaseService, tableName, jobName, status)
	toks := strings.Split(jobName, "-")
	msg := transcribe.EmailMessage{
		To:   toks[0],
		Body: jobName + " has completed.",
	}
	err = transcribe.PushEmailToQueue(sqsService, msg, queueUrl)
	return err

}

var databaseService *dynamodb.DynamoDB
var tableName string
var sqsService *sqs.SQS
var queueUrl string

func main() {

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	databaseService = dynamodb.New(sess)
	tableName = os.Getenv("TABLE_NAME")

	sqsService = sqs.New(sess)
	queueUrl = os.Getenv("EMAIL_QUEUE_URL")

	lambda.Start(HandleRequest)
}
