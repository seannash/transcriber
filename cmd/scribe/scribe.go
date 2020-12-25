package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	//	"github.com/USERNAME/simple-go-service/internal/something"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentity"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/transcribeservice"
)

func main() {
	//fmt.Println("There", something.Do())

	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{Region: aws.String("us-east-1"), Credentials: nil},
	})
	r, sess2 := Login(sess, "user0", "FUKyou42!")
	fmt.Println(r)
	// importyant token := *r.AuthenticationResult.IdToken
	//ans, err := GetRequest("https://h3ksw34ggi.execute-api.us-east-1.amazonaws.com/prod/job/2", token)
	//fmt.Println(ans, err)
	//urljsonbytes, err := PostRequest("https://h3ksw34ggi.execute-api.us-east-1.amazonaws.com/prod/job/1", token)
	//url = url[1 : len(url)-1]
	//var url string
	//err = json.Unmarshal(urljsonbytes, &url)
	//fmt.Println(url, err)
	//fmt.Println(url[0])
	//SendFile(url, "0001.mp4")
	// Set up a new s3manager client

	bucket := flag.String("bucket", "tim-training-thing", "The s3 bucket to upload to")
	filename := flag.String("filename", "", "The file to be uploaded")
	flag.Parse()

	uploader := s3manager.NewUploader(sess2)

	file, err := os.Open(*filename)

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	key := filepath.Base(file.Name())
	loc := "s3://" + *bucket + "/users/user0/" + key
	fmt.Println(loc, key)
	uploadOutput, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(*bucket),
		Key:    aws.String("users/user0/" + key),
		Body:   file,
	})
	fmt.Println(uploadOutput, err)
	//if err != nil {
	//	fmt.Println(err)
	//}
	fmt.Println("Done")
	os.Exit(0)

	transcriber := transcribeservice.New(sess)

	jobname := "4"
	mediaformat := "mp4"
	languagecode := "en-US"

	fmt.Println(uploadOutput, uploadOutput.Location)
	var media transcribeservice.Media
	media.MediaFileUri = &loc
	outputKey := "done/" + key + ".json"
	aa, err := transcriber.StartTranscriptionJob(&transcribeservice.StartTranscriptionJobInput{
		TranscriptionJobName: &jobname,
		Media:                &media,
		MediaFormat:          &mediaformat,
		LanguageCode:         &languagecode,
		OutputBucketName:     bucket,
		OutputKey:            &outputKey,
	})
	fmt.Println(aa, err)
	max_tries := 6000
	params := transcribeservice.GetTranscriptionJobInput{TranscriptionJobName: &jobname}
	for i := 0; i < max_tries; i += 1 {
		time.Sleep(time.Duration(10) * time.Second)
		j, err := transcriber.GetTranscriptionJob(&params)
		if err == nil {
			fmt.Println(j)
			if *j.TranscriptionJob.TranscriptionJobStatus == "COMPLETED" {
				turi := *j.TranscriptionJob.Transcript.TranscriptFileUri
				fmt.Println("URI: " + turi)
				//Download(turi, "out.json")
				break
			}
		}
	}
}

func GetRequest(url string, token string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal("Error reading request. ", err)
	}

	req.Header.Set("Auth", token)

	client := &http.Client{Timeout: time.Second * 10}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Error reading response. ", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Error reading body. ", err)
	}
	return string(body), nil
}

func PostRequest(url string, token string) ([]byte, error) {
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		log.Fatal("Error reading request. ", err)
	}

	req.Header.Set("Auth", token)

	client := &http.Client{Timeout: time.Second * 10}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Error reading response. ", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Error reading body. ", err)
	}
	return body, nil
}

/*
func Download(url string, fileout string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(fileout)
	if err != nil {
		return err
	}
	defer out.Close()

	var b := resp.Body
	var j interface()
	err := json.Unmarshal(b, &j)
	m:=j.([map[string]interface])

	_, err = io.Copy(out, resp.Body)
	return err
}
*/

func newfileUploadRequest(uri string, params map[string]string, paramName, path string) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}
	file.Close()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(paramName, fi.Name())
	if err != nil {
		return nil, err
	}
	part.Write(fileContents)

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	return http.NewRequest("PUT", uri, body)
}

func SendFile(url string, filename string) error {
	var extraParams map[string]string
	request, err := newfileUploadRequest(url, extraParams, "file", filename)
	if err != nil {
		log.Fatal(err)
	}
	client := &http.Client{}
	resp, err := client.Do(request)
	fmt.Println(resp, err)
	if err != nil && resp != nil && resp.Body != nil {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		fmt.Println("Response: ", string(body), err)
	}
	if err != nil {
		log.Fatal(err)
	}
	return err
}

func Login(sess *session.Session, user string, password string) (*cognitoidentityprovider.InitiateAuthOutput, *session.Session) {
	var apikey = "6kjlvu87ogi70h4qrqqj68mvr1"
	params := &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: aws.String("USER_PASSWORD_AUTH"),
		AuthParameters: map[string]*string{
			"USERNAME": aws.String(user),
			"PASSWORD": aws.String(password),
		},
		ClientId: aws.String(apikey),
	}
	cip := cognitoidentityprovider.New(sess)
	authResp, err := cip.InitiateAuth(params)
	fmt.Println(authResp, err)
	svc := cognitoidentity.New(sess)
	idRes, err := svc.GetId(&cognitoidentity.GetIdInput{
		IdentityPoolId: aws.String("us-east-1:acf87d88-718b-45bb-bb2d-93cb0f53a252"), //("us-east-1_BxAsOozif"),
		Logins: map[string]*string{
			"cognito-idp.us-east-1.amazonaws.com/us-east-1_BxAsOozif": authResp.AuthenticationResult.IdToken,
		},
	})
	credRes, err := svc.GetCredentialsForIdentity(&cognitoidentity.GetCredentialsForIdentityInput{
		IdentityId: idRes.IdentityId,
		Logins: map[string]*string{
			"cognito-idp.us-east-1.amazonaws.com/us-east-1_BxAsOozif": authResp.AuthenticationResult.IdToken,
		},
	})

	sess2, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
		Credentials: credentials.NewStaticCredentials(*credRes.Credentials.AccessKeyId,
			*credRes.Credentials.SecretKey,
			*credRes.Credentials.SessionToken)})

	return authResp, sess2
}
