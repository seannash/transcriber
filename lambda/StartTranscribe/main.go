package main

import (
	"context"
	"log"
	"time"

	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/transcribeservice"

	"example.com/transcribe/internal/database"
	"example.com/transcribe/internal/transcribe"
	"example.com/transcribe/internal/types"
)

func HandleRequest(ctx context.Context, events events.S3Event) (string, error) {
	for _, record := range events.Records {
		key := record.S3.Object.Key
		bucket := record.S3.Bucket.Name
		tokens := strings.Split(key, "/")
		loc := "s3://" + bucket + "/" + key
		tok := strings.Split(key, "/")
		jobName := transcribe.MakeJobId(tok[1], time.Now().Unix())
		rec := types.JobRecord{Job: jobName, User: tokens[1], JobStatus: "IN_PROGRESS", SourceURI: loc, ResultBucket: bucket, ResultKey: "done/" + key + ".json"}
		database.CreateRecord(databaseService, tableName, rec)
		transcribe.CallTranscribe(transcribeService, rec)
	}
	return events.Records[0].S3.Object.Key, nil
}

var databaseService *dynamodb.DynamoDB
var transcribeService *transcribeservice.TranscribeService
var tableName string = "transcriber"

func main() {

	tableName = "transcriber"

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	databaseService = dynamodb.New(sess)
	if databaseService == nil {
		log.Printf("Unable to create Dynamodb session\n")
		return
	}

	transcribeService = transcribeservice.New(sess)
	if transcribeService == nil {
		log.Printf("Unable to create Transcribe session\n")
		return
	}

	lambda.Start(HandleRequest)
}
