package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"example.com/transcribe/internal/cloud"
	"example.com/transcribe/internal/transcribe"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/transcribeservice"
)

type args struct {
	userName *string
	password *string
	bucket   *string
	fileName *string
	remote   *bool
	config   *string
	job      *string
}

func main() {
	var arg args

	transribeCommand := flag.NewFlagSet("transcribe", flag.ExitOnError)
	arg.fileName = transribeCommand.String("filename", "", "The file to be uploaded")
	arg.remote = transribeCommand.Bool("remote", false, "abc")

	listCommand := flag.NewFlagSet("list", flag.ExitOnError)

	getCommand := flag.NewFlagSet("get", flag.ExitOnError)
	arg.job = getCommand.String("job", "", "The job id")

	switch os.Args[1] {
	case "transcribe":
		transribeCommand.Parse(os.Args[2:])
	case "list":
		listCommand.Parse(os.Args[2:])
	case "get":
		getCommand.Parse(os.Args[2:])
	default:
		fmt.Printf("%q is not valid command.\n", os.Args[1])
		os.Exit(2)
	}
	config, err := LoadConfiguration("configuration.json")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if transribeCommand.Parsed() {
		if *arg.remote == true {
			DoRemote(arg, config)
		} else {
			DoLocal(arg, config)
		}
	}
	if listCommand.Parsed() {
		fmt.Println("Listing")
		ListJobs(arg, config)
	}
	if getCommand.Parsed() {
		fmt.Println("Get")
		GetJob(arg, config)
	}
	os.Exit(0)
}

func ListJobs(arg args, config Config) error {
	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{Region: aws.String(config.Region), Credentials: nil},
	})
	lparams := cloud.LoginParams{
		Region:        config.Region,
		ApiKey:        config.ApiKey,
		CognitoServer: config.CognitoServer,
		IdentityPool:  config.IdentityPool,
		UserName:      config.UserName,
		Password:      config.Password,
	}
	authRequestOutput, _, err := cloud.Login(sess, lparams)
	fmt.Println(authRequestOutput, err)
	token := *authRequestOutput.AuthenticationResult.IdToken
	data, err := cloud.GetRequest("https://z7fyh0rt5a.execute-api.us-east-1.amazonaws.com/prod/job/", token)
	fmt.Println(string(data), err)
	return err
}

func GetJob(arg args, config Config) error {
	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{Region: aws.String(config.Region), Credentials: nil},
	})
	lparams := cloud.LoginParams{
		Region:        config.Region,
		ApiKey:        config.ApiKey,
		CognitoServer: config.CognitoServer,
		IdentityPool:  config.IdentityPool,
		UserName:      config.UserName,
		Password:      config.Password,
	}
	authRequestOutput, _, _ := cloud.Login(sess, lparams)
	token := *authRequestOutput.AuthenticationResult.IdToken
	//fmt.Println(authRequestOutput, token)
	uri := "https://h3ksw34ggi.execute-api.us-east-1.amazonaws.com/prod/job/" + *arg.job
	fmt.Println(uri)
	data, err := cloud.GetRequest(uri, token)
	var url string
	json.Unmarshal(data, &url)
	fmt.Println("HI: ", url)
	return err
}

type Config struct {
	Region        string `json:"region"`
	ApiKey        string `json:"apiKey"`
	CognitoServer string `json:"cognitoServer"`
	IdentityPool  string `json:"identityPool"`
	UserName      string `json:"userName"`
	Password      string `json:"password"`
	Bucket        string `json:"bucket"`
}

func LoadConfiguration(file string) (Config, error) {
	var config Config
	configFile, err := os.Open(file)
	defer configFile.Close()
	if err != nil {
		fmt.Println(err.Error())
		return config, err
	}
	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)
	return config, nil
}

func DoRemote(arg args, config Config) error {
	fmt.Println("DoRemote")

	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{Region: aws.String(config.Region), Credentials: nil},
	})
	lparams := cloud.LoginParams{
		Region:        config.Region,
		ApiKey:        config.ApiKey,
		CognitoServer: config.CognitoServer,
		IdentityPool:  config.IdentityPool,
		UserName:      config.UserName,
		Password:      config.Password,
	}
	authRequestOutput, _, err := cloud.Login(sess, lparams)
	fmt.Println(authRequestOutput, err)
	token := *authRequestOutput.AuthenticationResult.IdToken
	data, err := cloud.PostRequest("https://z7fyh0rt5a.execute-api.us-east-1.amazonaws.com/prod/job/", token)
	sdata := string(data)
	fmt.Println("A", sdata, err)
	return err
	/*
		sess, err := session.NewSessionWithOptions(session.Options{
			Config: aws.Config{Region: aws.String(config.Region), Credentials: nil},
		})
		lparams := cloud.LoginParams{
			Region:        config.Region,
			ApiKey:        config.ApiKey,
			CognitoServer: config.CognitoServer,
			IdentityPool:  config.IdentityPool,
			UserName:      config.UserName,
			Password:      config.Password,
		}
		fmt.Println(lparams)
		_, sess2, _ := cloud.Login(sess, lparams)
		baseFileName := filepath.Base(*arg.fileName)
		key := "users/" + config.UserName + "/" + baseFileName
		loc := cloud.S3Location{
			Bucket: config.Bucket,
			Key:    key,
		}
		err = cloud.UploadFileToS3(sess2, *arg.fileName, loc)

		return err
	*/
}

func DoLocal(arg args, config Config) error {
	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{Region: aws.String(config.Region)},
	})
	if err != nil {
		return err
	}
	file, err := os.Open(*arg.fileName)
	if err != nil {
		return err
	}
	defer file.Close()
	baseFileName := filepath.Base(file.Name())

	//key := "todo/" + config.UserName + "/" + baseFileName
	key := "user/" + config.UserName + "/" + baseFileName
	loc := cloud.S3Location{
		Bucket: config.Bucket,
		Key:    key,
	}
	err = cloud.UploadFileToS3(sess, *arg.fileName, loc)

	return err

	transcriber := transcribeservice.New(sess)

	jobname := transcribe.MakeJobId("something", time.Now().Unix())
	mediaformat := "mp4"
	languagecode := "en-US"

	loc2 := "s3://" + config.Bucket + "/user/" + config.UserName + "/" + baseFileName
	var media transcribeservice.Media
	media.MediaFileUri = &loc2
	outputKey := "done/" + baseFileName + ".json"
	tparams := transcribeservice.StartTranscriptionJobInput{
		TranscriptionJobName: &jobname,
		Media:                &media,
		MediaFormat:          &mediaformat,
		LanguageCode:         &languagecode,
		OutputBucketName:     &config.Bucket,
		OutputKey:            &outputKey,
	}
	_, err = transcriber.StartTranscriptionJob(&tparams)
	if err != nil {
		return err
	}
	max_tries := 6000
	params := transcribeservice.GetTranscriptionJobInput{TranscriptionJobName: &jobname}
	for i := 0; i < max_tries; i += 1 {
		time.Sleep(time.Duration(10) * time.Second)
		j, err := transcriber.GetTranscriptionJob(&params)
		if err == nil {
			if *j.TranscriptionJob.TranscriptionJobStatus == "COMPLETED" {
				out, err := os.Create(*arg.fileName + ".json")
				if err != nil {
					return err
				}
				defer out.Close()
				downloader := s3manager.NewDownloader(sess)
				_, err = downloader.Download(out,
					&s3.GetObjectInput{
						Bucket: aws.String(config.Bucket),
						Key:    aws.String(outputKey),
					})
				return err
			}
		}
	}

	return nil
}
