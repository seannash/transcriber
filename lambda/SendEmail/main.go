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
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/ses"
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
		msg.To, err = transcribe.GetEmailFromUser(globalArea.COGNITO, globalArea.Pool, msg.To)

		err = transcribe.SesSend(globalArea.SES, globalArea.SendingAddress, msg.To, msg.Body)
		if err != nil {
			fmt.Println(err)
		}
	}
	return outerr
}

type GlobalArea struct {
	Sess           *session.Session
	Pool           string
	SendingAddress string
	SES            *ses.SES
	COGNITO        *cognitoidentityprovider.CognitoIdentityProvider
}

func SetGlobalArea(area GlobalArea) {
	globalArea = area
}

var globalArea GlobalArea

func main() {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	SetGlobalArea(GlobalArea{
		Sess:           sess,
		Pool:           os.Getenv("USER_POOL"),
		SendingAddress: os.Getenv("EMAIL_USER"),
		SES:            ses.New(sess),
		COGNITO:        cognitoidentityprovider.New(sess)})
	lambda.Start(HandleRequest)
}
