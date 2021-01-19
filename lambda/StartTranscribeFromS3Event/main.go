package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"example.com/transcribe/internal/transcribe"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/transcribeservice"
)

func HandleRequest(ctx context.Context, events events.S3Event) error {
	var err error
	for _, record := range events.Records {
		key := record.S3.Object.Key
		bucket := record.S3.Bucket.Name
		tokens := strings.Split(key, "/")
		loc := "s3://" + bucket + "/" + key
		tok := strings.Split(key, "/")
		jobName := transcribe.MakeJobId(tok[1], time.Now().Unix())
		rec := transcribe.JobRecord{
			Job:          jobName,
			User:         tokens[1],
			JobStatus:    "IN_PROGRESS",
			SourceURI:    loc,
			ResultBucket: bucket,
			ResultKey:    "done/" + key + ".json",
		}
		err = transcribe.CreateDatabaseRecord(databaseService, tableName, rec)
		err = transcribe.CallTranscribe(transcribeService, rec)
	}
	return err
}

var databaseService dynamodbiface.DynamoDBAPI
var transcribeService *transcribeservice.TranscribeService
var tableName string

func main() {

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	databaseService = dynamodb.New(sess)
	if databaseService == nil {
		log.Printf("Unable to create Dynamodb session\n")
		return
	}
	tableName = os.Getenv("TABLE_NAME")

	transcribeService = transcribeservice.New(sess)
	if transcribeService == nil {
		log.Printf("Unable to create Transcribe session\n")
		return
	}

	lambda.Start(HandleRequest)
}
