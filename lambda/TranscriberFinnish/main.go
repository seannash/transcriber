package main

import (
	"context"
	"encoding/json"
	"os"

	"example.com/transcribe/internal/database"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func HandleRequest(ctx context.Context, event events.CloudWatchEvent) error {
	mapStructure := &map[string]string{}
	_ = json.Unmarshal(event.Detail, mapStructure)
	jobName := (*mapStructure)["TranscriptionJobName"]
	status := (*mapStructure)["TranscriptionJobStatus"]
	err := database.SetStatus(databaseService, tableName, jobName, status)
	return err
}

var databaseService *dynamodb.DynamoDB
var tableName string

func main() {
	tableName = os.Getenv("TABLE_NAME")

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	databaseService = dynamodb.New(sess)
	lambda.Start(HandleRequest)
}
