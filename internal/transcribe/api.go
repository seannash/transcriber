package transcribe

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/sqs"
)

func PushEmailToQueue(svc *sqs.SQS, msg EmailMessage, queueUrl string) error {
	bytes, err := json.Marshal(msg)
	if err == nil {
		_, err := svc.SendMessage(&sqs.SendMessageInput{
			MessageBody: aws.String(string(bytes)),
			QueueUrl:    aws.String(queueUrl),
		})
		if err != nil {
			fmt.Println("Unable to send message to ", msg.To, " with body: ", msg.Body, "\n", err)
		}
	}
	return err
}

func SendEmail(sess *session.Session, pool string, msg EmailMessage, sender string) error {
	cognito := cognitoidentityprovider.New(sess)
	userEmailAddress, err := GetEmailFromUser(cognito, pool, msg.To)
	fmt.Println(userEmailAddress, err)
	if err != nil {
		fmt.Println("Cannot get email address for : ", msg.To, "\n", err.Error())
		return err
	}
	fmt.Println("Sending address ", sender)
	err = SesSend(sess, sender, userEmailAddress, msg.Body)
	return err
}
