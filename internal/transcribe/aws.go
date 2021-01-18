package transcribe

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/service/transcribeservice"
)

const (
	Subject = "Transcriber Job Finnish"
	CharSet = "UTF-8"
)

func SesSend(sess *session.Session, sender string, recipient string, body string) error {
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

func CallTranscribe(tranService *transcribeservice.TranscribeService, record JobRecord) error {

	mediaformat := "mp4"
	languagecode := "en-US"

	var media transcribeservice.Media
	media.MediaFileUri = &record.SourceURI

	params := transcribeservice.StartTranscriptionJobInput{
		TranscriptionJobName: &record.Job,
		Media:                &media,
		MediaFormat:          &mediaformat,
		LanguageCode:         &languagecode,
		OutputBucketName:     &record.ResultBucket,
		OutputKey:            &record.ResultKey,
	}
	fmt.Println(params)
	_, err := tranService.StartTranscriptionJob(&params)
	if err != nil {
		fmt.Println(err.Error())
	}
	return err
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
