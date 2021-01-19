package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"

	"example.com/transcribe/internal/transcribe"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

const (
	ErrorMethodNotAllowed = "method Not allowed"
	ErrorNotImplemented   = "not implemented"
)

var (
	dynaClient    dynamodbiface.DynamoDBAPI
	s3Service     s3iface.S3API
	tableName     string
	projectBucket string
)

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
	// Get Parameters
	user, err := getUserFromRequest(req)
	if err != nil {
		return apiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
	}
	area, _ := req.PathParameters["area"]
	id := getIdFromRequest(req)
	// Call function to create a response boy
	var body interface{}
	if area == "job" && id != "" {
		body, err = transcribe.GetJobLocation(dynaClient, s3Service, tableName, id)
	} else if area == "job" && id == "" {
		body, err = transcribe.ListJobs(user, tableName, dynaClient)
	} else if area == "upload" && id != "" {
		body, err = transcribe.GetUploadUri(s3Service, projectBucket, user, id)
	} else {
		err = errors.New("MethodNotAllowed")
	}
	// Return Response with body or error
	if err != nil {
		return apiResponse(http.StatusBadRequest, ErrorBody{aws.String(err.Error())})
	}
	return apiResponse(http.StatusOK, body)
}

func getUserFromRequest(req events.APIGatewayProxyRequest) (string, error) {
	p := req.RequestContext.Authorizer
	claims := p["claims"]
	if claims == nil {
		return "", errors.New("No claims")
	}
	userBlob := claims.(map[string]interface{})["cognito:username"]
	user, ok := userBlob.(string)
	if !ok {
		return "", errors.New("No username")
	}
	return user, nil
}

func getIdFromRequest(req events.APIGatewayProxyRequest) string {
	idRaw, idExists := req.PathParameters["id"]
	var id string
	if idExists {
		id, _ = url.QueryUnescape(idRaw)
		return id
	}
	return ""
}

func optionsResponse() (*events.APIGatewayProxyResponse, error) {
	resp := events.APIGatewayProxyResponse{
		Headers: map[string]string{
			"Content-Type":                 "application/json",
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "POST, GET, OPTIONS, PUT, DELETE",
			"Access-Control-Allow-Headers": "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization"}}
	resp.StatusCode = 200
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

type ErrorBody struct {
	ErrorMsg *string `json:"error,omitempty"`
}
