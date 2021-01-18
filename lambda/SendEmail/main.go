package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"example.com/transcribe/internal/transcribe"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
)

func HandleRequest(ctx context.Context, events events.SQSEvent) error {
	var outerr error
	for _, record := range events.Records {
		body := record.Body
		var msg transcribe.EmailMessage
		err := json.Unmarshal([]byte(body), &msg)
		if err != nil {
			fmt.Println("Cannot parse: ", body, "\n", err.Error())
			continue
		}
		err = transcribe.SendEmail(sess, pool, msg, sendingAddress)
		if err != nil {
			outerr = err
			fmt.Println()
		}
	}
	return outerr
}

var sess *session.Session
var pool string
var sendingAddress string

func main() {
	pool = os.Getenv("USER_POOL")
	sendingAddress = os.Getenv("EMAIL_USER")
	sess = session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	lambda.Start(HandleRequest)
}
