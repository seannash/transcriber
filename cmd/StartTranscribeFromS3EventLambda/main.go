package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/transcribeservice"
)

type StartTranscriptionJob interface {
	StartTranscriptionJob(input *transcribeservice.StartTranscriptionJobInput) (*transcribeservice.StartTranscriptionJobOutput, error)
}

func CallTranscribe(svc StartTranscriptionJob, record JobRecord) error {

	mediaformat := "mp4"
	languagecode := "en-US"

	var media transcribeservice.Media
	media.MediaFileUri = &record.SourceURI

	params := transcribeservice.StartTranscriptionJobInput{
		TranscriptionJobName: &record.Job,
		Media:                &media,
		MediaFormat:          &mediaformat,
		LanguageCode:         &languagecode,
		OutputBucketName:     &record.ResultBucket,
		OutputKey:            &record.ResultKey,
	}
	_, err := svc.StartTranscriptionJob(&params)
	if err != nil {
		fmt.Println(err.Error())
	}
	return err
}

type JobRecord struct {
	Job          string `json:"job"`
	User         string `json:"user"`
	JobStatus    string `json:"job_status"`
	SourceURI    string `json:"source_uri"`
	ResultBucket string `json:"result_bucket"`
	ResultKey    string `json:"result_key"`
}

type DdbPutItem interface {
	PutItem(*dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
}

func CreateDatabaseRecord(dbSvc DdbPutItem, table string, record JobRecord) error {
	av, err := dynamodbattribute.MarshalMap(record)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(table),
	}

	_, err = dbSvc.PutItem(input)
	if err != nil {
		fmt.Println(err.Error())
	}

	return err
}

type JobFunc func(record JobRecord) error

type LambdaContext struct {
	startTranscribe JobFunc
	createRecord    JobFunc
}

func MakeJobId(base string, num int64) string {
	return fmt.Sprintf("%s-%d", base, num)
}

func (lc LambdaContext) HandleRequest(ctx context.Context, events events.S3Event) error {
	var err error
	for _, record := range events.Records {
		key := record.S3.Object.Key
		bucket := record.S3.Bucket.Name
		tokens := strings.Split(key, "/")
		loc := "s3://" + bucket + "/" + key
		tok := strings.Split(key, "/")
		jobName := MakeJobId(tok[1], time.Now().Unix())
		rec := JobRecord{
			Job:          jobName,
			User:         tokens[1],
			JobStatus:    "IN_PROGRESS",
			SourceURI:    loc,
			ResultBucket: bucket,
			ResultKey:    "done/" + key + ".json",
		}
		err = lc.createRecord(rec)
		err = lc.startTranscribe(rec)
	}
	return err
}

func main() {

	tableName := os.Getenv("TABLE_NAME")

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	databaseService := dynamodb.New(sess)
	if databaseService == nil {
		log.Printf("Unable to create Dynamodb session\n")
		return
	}

	transcribeService := transcribeservice.New(sess)
	if transcribeService == nil {
		log.Printf("Unable to create Transcribe session\n")
		return
	}

	lambdaContext := LambdaContext{
		createRecord: func(record JobRecord) error {
			return CreateDatabaseRecord(databaseService, tableName, record)
		},
		startTranscribe: func(record JobRecord) error {
			return CallTranscribe(transcribeService, record)
		},
	}

	lambda.Start(func(ctx context.Context, events events.S3Event) error {
		return lambdaContext.HandleRequest(ctx, events)
	})
}
