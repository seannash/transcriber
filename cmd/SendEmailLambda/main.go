package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/ses"
)

const (
	Subject = "Transcriber Job Finnish"
	CharSet = "UTF-8"
)

type SesSendEmail interface {
	SendEmail(input *ses.SendEmailInput) (*ses.SendEmailOutput, error)
}

func Send(svc SesSendEmail, sender string, recipient string, body string) error {
	// Assemble the email.
	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			CcAddresses: []*string{},
			ToAddresses: []*string{
				aws.String(recipient),
			},
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Text: &ses.Content{
					Charset: aws.String(CharSet),
					Data:    aws.String(body),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String(CharSet),
				Data:    aws.String(Subject),
			},
		},
		Source: aws.String(sender),
	}

	result, err := svc.SendEmail(input)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ses.ErrCodeMessageRejected:
				fmt.Println(ses.ErrCodeMessageRejected, aerr.Error())
			case ses.ErrCodeMailFromDomainNotVerifiedException:
				fmt.Println(ses.ErrCodeMailFromDomainNotVerifiedException, aerr.Error())
			case ses.ErrCodeConfigurationSetDoesNotExistException:
				fmt.Println(ses.ErrCodeConfigurationSetDoesNotExistException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
			fmt.Println(err.Error())
		}

		return err
	}

	fmt.Println("Email Sent to address: " + recipient)
	fmt.Println(result)
	return nil
}

type CognitoGetUser interface {
	AdminGetUser(input *cognitoidentityprovider.AdminGetUserInput) (*cognitoidentityprovider.AdminGetUserOutput, error)
}

func GetEmailFromUser(svc CognitoGetUser, pool string, user string) (string, error) {
	record, err := svc.AdminGetUser(&cognitoidentityprovider.AdminGetUserInput{
		UserPoolId: aws.String(pool),
		Username:   aws.String(user),
	})
	if err != nil {
		return "", err
	}
	for _, v := range record.UserAttributes {
		if *v.Name == "email" {
			return *v.Value, nil
		}
	}
	return "", errors.New("email not found")
}

type EmailMessage struct {
	To   string `json:"to"`
	Body string `json:"body"`
}

type SendEmailFunc func(sender string, recipient string, body string) error
type GetUserFunc func(user string) (string, error)

func InternalHandleRequest(ctx context.Context, events events.SQSEvent, getUser GetUserFunc, sendEmail SendEmailFunc) error {
	var outerr error
	for _, record := range events.Records {
		body := record.Body
		var msg EmailMessage
		err := json.Unmarshal([]byte(body), &msg)
		if err != nil {
			fmt.Println("Cannot parse: ", body, "\n", err.Error())
			continue
		}
		msg.To, err = getUser(msg.To)
		if err != nil {
			fmt.Println("Cannot get email for user: ", msg.To)
			continue
		}
		err = sendEmail(globalArea.SendingAddress, msg.To, msg.Body)
		if err != nil {
			fmt.Println(err)
		}
	}
	return outerr
}

func HandleRequest(ctx context.Context, events events.SQSEvent) error {
	getUser := func(user string) (string, error) {
		return GetEmailFromUser(globalArea.COGNITO, globalArea.Pool, user)
	}
	sendEmail := func(sender string, recipient string, body string) error {
		return Send(globalArea.SES, sender, recipient, body)
	}
	return InternalHandleRequest(ctx, events, getUser, sendEmail)
}

type GlobalArea struct {
	Sess           *session.Session
	Pool           string
	SendingAddress string
	SES            *ses.SES
	COGNITO        *cognitoidentityprovider.CognitoIdentityProvider
}

var globalArea GlobalArea

func main() {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	globalArea = GlobalArea{
		Sess:           sess,
		Pool:           os.Getenv("USER_POOL"),
		SendingAddress: os.Getenv("EMAIL_USER"),
		SES:            ses.New(sess),
		COGNITO:        cognitoidentityprovider.New(sess)}
	lambda.Start(HandleRequest)
}
