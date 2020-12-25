package main

import (
	"context"

	"example.com/transcribe/internal/database"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"

	"github.com/aws/aws-lambda-go/lambda"
)

type TranscribFinnishEvent struct {
	JobName   string `json:"jobName"`
	JobStatus string `json:"jobStatus"`
}

func HandleRequest(ctx context.Context, event TranscribFinnishEvent) error {
	err := database.SetStatus(databaseService, tableName, event.JobName, event.JobStatus)
	return err
}

var databaseService *dynamodb.DynamoDB
var tableName string

func main() {
	tableName = "transcriber"

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	databaseService = dynamodb.New(sess)
	lambda.Start(HandleRequest)
}
