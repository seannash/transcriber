package database

import (
	"errors"
	"fmt"

	"example.com/transcribe/internal/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

func CreateRecord(dbSvc *dynamodb.DynamoDB, table string, record types.JobRecord) error {
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

func GetRecord(svc *dynamodb.DynamoDB, table string, job string) (types.JobRecord, error) {
	const tableName = "transcriber"

	result, err := svc.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"job": {
				S: aws.String(job),
			},
		},
	})
	var rec types.JobRecord
	if err != nil {
		fmt.Println(result)
	}
	if result.Item == nil {
		msg := "Could not find '" + job + ""
		return rec, errors.New(msg)
	}

	err = dynamodbattribute.UnmarshalMap(result.Item, &rec)
	if err != nil {
		panic(fmt.Sprintf("Failed to unmarshal Record, %v", err))
	}

	return rec, err
}

func SetStatus(dbService dynamodbiface.DynamoDBAPI, table string, job string, status string) error {
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
