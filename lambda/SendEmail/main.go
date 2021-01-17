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

func SendEmail(sess *session.Session, sender string, recipient string, body string) error {
	// Assemble the email.
	svc := ses.New(sess)
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
	fmt.Println(input)

	// Attempt to send the email.
	result, err := svc.SendEmail(input)

	// Display error messages if they occur.
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
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}

		return err
	}

	fmt.Println("Email Sent to address: " + recipient)
	fmt.Println(result)
	return nil
}

func GetEmailFromUser(svc *cognitoidentityprovider.CognitoIdentityProvider, pool string, user string) (string, error) {
	record, err := svc.AdminGetUser(&cognitoidentityprovider.AdminGetUserInput{
		UserPoolId: aws.String(pool),
		Username:   aws.String(user),
	})
	if err != nil {
		return "", err
	}
	fmt.Println(record)
	for _, v := range record.UserAttributes {
		fmt.Println(*v.Name, *v.Value)
		if *v.Name == "email" {
			return *v.Value, nil
		}
	}
	return "", errors.New("Email Not Found")
}

type EmailMessage struct {
	To   string `json:"to"`
	Body string `json:"body"`
}

func HandleRequest(ctx context.Context, events events.SQSEvent) {
	for _, record := range events.Records {
		body := record.Body
		var msg EmailMessage
		err := json.Unmarshal([]byte(body), &msg)
		if err != nil {
			fmt.Println("Cannot parse: ", body, "\n", err.Error())
			continue
		}
		cognito := cognitoidentityprovider.New(sess)
		userEmailAddress, err := GetEmailFromUser(cognito, pool, msg.To)
		fmt.Println(userEmailAddress, err)
		if err != nil {
			fmt.Println("Cannot get email address for : ", msg.To, "\n", err.Error())
			continue
		}
		fmt.Println("Sending address ", sendingAddress)
		SendEmail(sess, sendingAddress, userEmailAddress, msg.Body)
		fmt.Printf("[%s] Email:  %s Message = %s \n", record.EventSource, sendingAddress, msg.Body)
	}
}

var sess *session.Session
var pool string
var sendingAddress string

func main() {
	pool = os.Getenv("USER_POOL")
	fmt.Println(pool)
	sendingAddress = os.Getenv("EMAIL_USER")
	sess = session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	lambda.Start(HandleRequest)
}
