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

	"example.com/transcribe/internal/transcribe"
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
var projectBucket string

func main() {
	tableName = os.Getenv("TABLE_NAME")
	region := os.Getenv("AWS_REGION")
	projectBucket = os.Getenv("PROJECT_BUCKET")
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

func optionsResponse() (*events.APIGatewayProxyResponse, error) {
	resp := events.APIGatewayProxyResponse{
		Headers: map[string]string{
			"Content-Type":                 "application/json",
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "POST, GET, OPTIONS, PUT, DELETE",
			"Access-Control-Allow-Headers": "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization"}}
	resp.StatusCode = 200

	//stringBody, _ := json.Marshal(body)
	//resp.Body = string(stringBody)
	return &resp, nil

}

func apiResponse(status int, body interface{}) (*events.APIGatewayProxyResponse, error) {
	resp := events.APIGatewayProxyResponse{
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*"}}
	resp.StatusCode = status

	if body != nil {
		stringBody, _ := json.Marshal(body)
		resp.Body = string(stringBody)
	}
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
	recBody, _ := json.Marshal(req)
	log.Print(string(recBody))
	switch req.HTTPMethod {
	case "GET":
		return HandlerGet(req, tableName, dynaClient)
	case "OPTIONS":
		return optionsResponse()
	default:
		return apiResponse(http.StatusMethodNotAllowed, ErrorMethodNotAllowed)
	}
}

func HandlerGet(req events.APIGatewayProxyRequest, tableName string, dynaClient dynamodbiface.DynamoDBAPI) (
	*events.APIGatewayProxyResponse,
	error,
) {
	log.Print("Identity: ", req.RequestContext)
	p := req.RequestContext.Authorizer // ["claims"]["cognito:username"]
	claims := p["claims"]
	if claims == nil {
		return apiResponse(200, "No claims")
	}
	userBlob := claims.(map[string]interface{})["cognito:username"]
	user := userBlob.(string)

	// user, _ := req.PathParameters["user"]
	area, _ := req.PathParameters["area"]
	idRaw, idExists := req.PathParameters["id"]
	var id string
	if idExists {
		id, _ = url.QueryUnescape(idRaw)
	}
	if area == "job" && idExists {
		// Single Mode
		result, err := GetJob(id, tableName, dynaClient)
		if err != nil {
			return apiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
		}
		if err != nil {
			fmt.Println(result)
			return apiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
		}
		resultBucket := result.ResultBucket
		resultKey := result.ResultKey
		fmt.Println("Key", result.ResultKey)
		signedURI, err := makeSignedURI(s3Service, resultBucket, resultKey)
		return apiResponse(http.StatusOK, signedURI)
	} else if area == "job" && !idExists {
		// List of user's jobs
		result, err := ListJobs(user, tableName, dynaClient)
		if err != nil {
			return apiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
		}
		return apiResponse(http.StatusOK, result)
	} else if area == "upload" && idExists {
		reqo, _ := s3Service.PutObjectRequest(&s3.PutObjectInput{
			Bucket: aws.String(projectBucket),
			Key:    aws.String("users/" + user + "/" + id),
		})
		urlStr, err := reqo.Presign(15 * time.Minute)
		if err != nil {
			log.Println("Failed to sign request", err)
		}
		return apiResponse(http.StatusOK, urlStr)
	}

	return apiResponse(http.StatusMethodNotAllowed, ErrorMethodNotAllowed)
}

func GetJob(job string, tableName string, dynaClient dynamodbiface.DynamoDBAPI) (*transcribe.JobRecord, error) {
	fmt.Println("GetJob")
	result, err := dynaClient.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"job": {
				S: aws.String(job),
			},
		},
	})
	item := new(transcribe.JobRecord)
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

func ListJobs(user string, table string, dynaClient dynamodbiface.DynamoDBAPI) (*[]transcribe.JobRecord, error) {
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

	items := new([]transcribe.JobRecord)
	if resp.Items != nil {
		err = dynamodbattribute.UnmarshalListOfMaps(resp.Items, &items)
	}

	return items, nil
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

func makeSignedPutURI(s3Service s3iface.S3API, bucket string, key string) (string, error) {

	reqo, _ := s3Service.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	uri, err := reqo.Presign(15 * time.Minute)

	if err != nil {
		log.Println("Failed to sign request", err)
	}

	return uri, err

}
