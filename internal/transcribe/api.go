package transcribe

import (
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

func GetJobLocation(DB dynamodbiface.DynamoDBAPI, S3 s3iface.S3API, table string, id string) (string, error) {
	result, err := GetJob(id, table, DB)
	if err != nil {
		return "", err
	}
	resultBucket := result.ResultBucket
	resultKey := result.ResultKey
	return MakeSignedURI(S3, resultBucket, resultKey)
}

func GetUploadUri(S3 s3iface.S3API, bucket string, user string, id string) (string, error) {
	reqo, _ := S3.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("users/" + user + "/" + id),
	})
	urlStr, err := reqo.Presign(15 * time.Minute)
	return urlStr, err
}

func GetJob(job string, tableName string, dynaClient dynamodbiface.DynamoDBAPI) (*JobRecord, error) {
	fmt.Println("GetJob")
	result, err := dynaClient.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"job": {
				S: aws.String(job),
			},
		},
	})
	item := new(JobRecord)
	if err != nil {
		fmt.Println(result)
		return item, errors.New("failed")
	}
	err = dynamodbattribute.UnmarshalMap(result.Item, item)
	if err != nil {
		return nil, errors.New("ErrorFailedToUnmarshalRecord")
	}
	return item, nil
}

func ListJobs(user string, table string, dynaClient dynamodbiface.DynamoDBAPI) (*[]JobRecord, error) {
	fmt.Println("ListJob", user)
	params := &dynamodb.QueryInput{
		TableName:              aws.String(table),
		IndexName:              aws.String("user-index"),
		KeyConditionExpression: aws.String("#user = :user"),
		ExpressionAttributeNames: map[string]*string{
			"#user": aws.String("user"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":user": {
				S: aws.String(user),
			},
		},
	}

	resp, err := dynaClient.Query(params)
	if err != nil {
		fmt.Printf("ERROAR: %v\n", err.Error())
		return nil, err
	}

	fmt.Println(resp)

	items := new([]JobRecord)
	if resp.Items != nil {
		err = dynamodbattribute.UnmarshalListOfMaps(resp.Items, &items)
	}

	return items, nil
}
