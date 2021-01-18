package transcribe

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/service/sqs"
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

type LoginParams struct {
	ApiKey   string
	UserName string
	Password string
}

func Login(sess *session.Session, lparams LoginParams) (*cognitoidentityprovider.InitiateAuthOutput, error) {

	params := &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: aws.String("USER_PASSWORD_AUTH"),
		AuthParameters: map[string]*string{
			"USERNAME": aws.String(lparams.UserName),
			"PASSWORD": aws.String(lparams.Password),
		},
		ClientId: aws.String(lparams.ApiKey),
	}

	cip := cognitoidentityprovider.New(sess)
	authResp, err := cip.InitiateAuth(params)
	return authResp, err
}

type S3Location struct {
	Bucket string
	Key    string
}

func UploadFileToS3(sess *session.Session, fileName string, loc S3Location) error {
	uploader := s3manager.NewUploader(sess)

	file, err := os.Open(fileName)

	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(loc.Bucket),
		Key:    aws.String(loc.Key),
		Body:   file,
	})
	return err
}

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

func MakeSignedURI(s3Service s3iface.S3API, bucket string, key string) (string, error) {

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

func MakeSignedPutURI(s3Service s3iface.S3API, bucket string, key string) (string, error) {

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
