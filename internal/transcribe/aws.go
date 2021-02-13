package transcribe

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/transcribeservice"
)

const (
	Subject = "Transcriber Job Finnish"
	CharSet = "UTF-8"
)

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
