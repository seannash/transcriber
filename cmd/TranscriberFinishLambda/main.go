package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

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

type SqsSendMessage interface {
	SendMessage(input *sqs.SendMessageInput) (*sqs.SendMessageOutput, error)
}

func SendMessageToSqs(svc SqsSendMessage, queueUrl string, msg EmailMessage) error {
	bytes, err := json.Marshal(msg)
	if err == nil {
		_, err := svc.SendMessage(&sqs.SendMessageInput{
			MessageBody: aws.String(string(bytes)),
			QueueUrl:    aws.String(queueUrl),
		})
		if err != nil {
			fmt.Println("Unable to send message to ", msg.To, " with body: ", msg.Body, "\n", err)
		}
	}
	return err
}

type DynamoDbUpdateItem interface {
	UpdateItem(*dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error)
}

func SetDatabaseRecordStatus(dbService DynamoDbUpdateItem, table string, job string, status string) error {
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":s": {
				S: aws.String(status),
			},
		},
		TableName: aws.String(table),
		Key: map[string]*dynamodb.AttributeValue{
			"job": {
				S: aws.String(job),
			},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String("set job_status = :s"),
	}
	_, err := dbService.UpdateItem(input)
	return err
}

type UpdateJobStatusFunc func(jobName string, status string) error
type PushEmailToQueueFunc func(msg EmailMessage) error

type LambdaContext struct {
	updateJobStatus  UpdateJobStatusFunc
	pushEmailToQueue PushEmailToQueueFunc
}

func (lc *LambdaContext) HandleRequest(ctx context.Context, event events.CloudWatchEvent) error {
	mapStructure := &map[string]string{}
	_ = json.Unmarshal(event.Detail, mapStructure)
	jobName := (*mapStructure)["TranscriptionJobName"]
	status := (*mapStructure)["TranscriptionJobStatus"]
	_ = lc.updateJobStatus(jobName, status)
	toks := strings.Split(jobName, "-")
	msg := EmailMessage{
		To:   toks[0],
		Body: jobName + " has completed.",
	}
	err := lc.pushEmailToQueue(msg)
	return err
}

func main() {

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	databaseService := dynamodb.New(sess)
	tableName := os.Getenv("TABLE_NAME")

	sqsService := sqs.New(sess)
	queueUrl := os.Getenv("EMAIL_QUEUE_URL")

	lambdaContext := LambdaContext{
		pushEmailToQueue: func(msg EmailMessage) error {
			return SendMessageToSqs(sqsService, queueUrl, msg)
		},
		updateJobStatus: func(jobName string, status string) error {
			return SetDatabaseRecordStatus(databaseService, tableName, jobName, status)
		},
	}
	lambda.Start(func(ctx context.Context, event events.CloudWatchEvent) error {
		return lambdaContext.HandleRequest(ctx, event)
	})
}
