package cloud

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

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
