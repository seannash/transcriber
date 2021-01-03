package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"example.com/transcribe/internal/types"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

var (
	dynaClient dynamodbiface.DynamoDBAPI
	s3Service  s3iface.S3API
)

var (
	ErrorFailedToUnmarshalRecord = "failed to unmarshal record"
	ErrorFailedToFetchRecord     = "failed to fetch record"
	ErrorInvalidUserData         = "invalid user data"
	ErrorInvalidEmail            = "invalid email"
	ErrorCouldNotDeleteItem      = "could not delete item"
	ErrorCouldNotDynamoPutItem   = "could not dynamo put item error"
	ErrorUserAlreadyExists       = "user.User already exists"
	ErrorUserDoesNotExists       = "user.User does not exist"
)

var tableName string

func main() {
	tableName = os.Getenv("TABLE_NAME")
	region := os.Getenv("AWS_REGION")
	awsSession, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)
	if err != nil {
		return
	}
	dynaClient = dynamodb.New(awsSession)
	s3Service = s3.New(awsSession)
	lambda.Start(handler)
}

func apiResponse(status int, body interface{}) (*events.APIGatewayProxyResponse, error) {
	resp := events.APIGatewayProxyResponse{Headers: map[string]string{"Content-Type": "application/json"}}
	resp.StatusCode = status

	stringBody, _ := json.Marshal(body)
	resp.Body = string(stringBody)
	return &resp, nil
}

var (
	ErrorMethodNotAllowed = "method Not allowed"
	ErrorNotImplemented   = "not implemented"
)

type ErrorBody struct {
	ErrorMsg *string `json:"error,omitempty"`
}

func handler(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	fmt.Println(req)
	switch req.HTTPMethod {
	case "GET":
		return HandlerGet(req, tableName, dynaClient)
	case "POST":
		return HandlerPost(req, tableName, dynaClient)
	default:
		return apiResponse(http.StatusMethodNotAllowed, ErrorMethodNotAllowed)
	}
}

func HandlerGet(req events.APIGatewayProxyRequest, tableName string, dynaClient dynamodbiface.DynamoDBAPI) (
	*events.APIGatewayProxyResponse,
	error,
) {
	fmt.Println("Identity: ", req.RequestContext)
	p := req.RequestContext.Authorizer // ["claims"]["cognito:username"]
	claims := p["claims"]
	userBlob := claims.(map[string]interface{})["cognito:username"]
	user := userBlob.(string)

	rawJob, found := req.PathParameters["id"]
	if found {
		// Single Mode
		job, err := url.QueryUnescape(rawJob)
		result, err := GetJob(job, tableName, dynaClient)
		if err != nil {
			return apiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
		}
		//item := new(types.JobRecord)
		if err != nil {
			fmt.Println(result)
			return apiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
		}
		resultBucket := result.ResultBucket
		resultKey := result.ResultKey
		signedURI, err := makeSignedURI(s3Service, resultBucket, resultKey)
		return apiResponse(http.StatusOK, signedURI)
	}
	// List of user's jobs
	result, err := ListJobs(user, tableName, dynaClient)
	if err != nil {
		return apiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
	}
	return apiResponse(http.StatusOK, result)
}

func GetJob(job string, tableName string, dynaClient dynamodbiface.DynamoDBAPI) (*types.JobRecord, error) {
	fmt.Println("GetJob")
	result, err := dynaClient.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"job": {
				S: aws.String(job),
			},
		},
	})
	item := new(types.JobRecord)
	if err != nil {
		fmt.Println(result)
		return item, errors.New("failed")
	}
	err = dynamodbattribute.UnmarshalMap(result.Item, item)
	if err != nil {
		return nil, errors.New(ErrorFailedToUnmarshalRecord)
	}
	return item, nil
}

func ListJobs(user string, table string, dynaClient dynamodbiface.DynamoDBAPI) (*[]types.JobRecord, error) {
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

	items := new([]types.JobRecord)
	err = dynamodbattribute.UnmarshalListOfMaps(resp.Items, &items)

	return items, nil
}

func HandlerPost(req events.APIGatewayProxyRequest, tableName string, dynaClient dynamodbiface.DynamoDBAPI) (
	*events.APIGatewayProxyResponse,
	error,
) {
	p := req.RequestContext.Authorizer // ["claims"]["cognito:username"]
	claims := p["claims"]
	userBlob := claims.(map[string]interface{})["cognito:username"]
	user := userBlob.(string)

	reqo, _ := s3Service.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String("transcriber"),
		Key:    aws.String("users/" + user + "/" + "1"),
	})
	urlStr, err := reqo.Presign(15 * time.Minute)

	if err != nil {
		log.Println("Failed to sign request", err)
	}

	return apiResponse(http.StatusOK, urlStr)
}

func makeSignedURI(s3Service s3iface.S3API, bucket string, key string) (string, error) {

	reqo, _ := s3Service.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	uri, err := reqo.Presign(15 * time.Minute)

	if err != nil {
		log.Println("Failed to sign request", err)
	}

	return uri, err

}
