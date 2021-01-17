package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"example.com/transcribe/internal/database"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/sqs"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type EmailMessage struct {
	To   string `json:"to"`
	Body string `json:"body"`
}

func HandleRequest(ctx context.Context, event events.CloudWatchEvent) error {
	mapStructure := &map[string]string{}
	_ = json.Unmarshal(event.Detail, mapStructure)
	jobName := (*mapStructure)["TranscriptionJobName"]
	status := (*mapStructure)["TranscriptionJobStatus"]
	err := database.SetStatus(databaseService, tableName, jobName, status)
	msg := EmailMessage{
		To:   "bubba",
		Body: jobName + " has completed.",
	}
	bytes, err := json.Marshal(msg)
	if err == nil {
		_, err := sqsService.SendMessage(&sqs.SendMessageInput{
			//DelaySeconds: aws.Int64(10),
			MessageBody: aws.String(string(bytes)),
			QueueUrl:    aws.String(queueUrl),
		})
		if err != nil {
			fmt.Println("Unable to send message to ", msg.To, " with body: ", msg.Body, "\n", err)
		}
	}

	return err
}

var databaseService *dynamodb.DynamoDB
var tableName string
var sqsService *sqs.SQS
var queueUrl string

func main() {
	tableName = os.Getenv("TABLE_NAME")

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	databaseService = dynamodb.New(sess)

	sqsService = sqs.New(sess)
	queueUrl = os.Getenv("EMAIL_QUEUE_URL")

	lambda.Start(HandleRequest)
}
