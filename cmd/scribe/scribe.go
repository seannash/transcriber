package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"example.com/transcribe/internal/cloud"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

type args struct {
	userName *string
	password *string
	bucket   *string
	fileName *string
	config   *string
	job      *string
}

func main() {
	var arg args

	transribeCommand := flag.NewFlagSet("transcribe", flag.ExitOnError)
	arg.fileName = transribeCommand.String("filename", "", "The file to be uploaded")
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
		DoRemote(arg, config)
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
		Config: aws.Config{Credentials: nil},
	})
	lparams := cloud.LoginParams{
		ApiKey:   config.ApiKey,
		UserName: config.UserName,
		Password: config.Password,
	}
	authRequestOutput, err := cloud.Login(sess, lparams)
	fmt.Println(authRequestOutput, err)
	token := *authRequestOutput.AuthenticationResult.IdToken
	url := config.Api + "/transcribe/" + config.UserName + "/job"
	data, err := cloud.GetRequest(url, token)
	fmt.Println(string(data), err)
	return err
}

func GetJob(arg args, config Config) error {
	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{Credentials: nil},
	})
	lparams := cloud.LoginParams{
		ApiKey:   config.ApiKey,
		UserName: config.UserName,
		Password: config.Password,
	}
	authRequestOutput, _ := cloud.Login(sess, lparams)
	token := *authRequestOutput.AuthenticationResult.IdToken
	uri := config.Api + "/transcribe/" + config.UserName + "/job/" + *arg.job
	fmt.Println(uri)
	data, err := cloud.GetRequest(uri, token)
	fmt.Println(string(data), err)
	var url string
	json.Unmarshal(data, &url)
	json, _ := cloud.GetString(url)
	fmt.Println(json)
	return err
}

type Config struct {
	ApiKey   string `json:"apiKey"`
	UserName string `json:"userName"`
	Password string `json:"password"`
	Api      string `json:"api"`
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
		Config: aws.Config{Credentials: nil},
	})
	lparams := cloud.LoginParams{
		ApiKey:   config.ApiKey,
		UserName: config.UserName,
		Password: config.Password,
	}
	authRequestOutput, err := cloud.Login(sess, lparams)
	fmt.Println(authRequestOutput, err)
	token := *authRequestOutput.AuthenticationResult.IdToken
	user := config.UserName
	file := *arg.fileName
	api := config.Api
	url := api + "/transcribe/" + user + "/upload/" + file
	fmt.Println(url)
	data, err := cloud.GetRequest(url, token)
	var reqURL string
	json.Unmarshal(data, &reqURL)
	fmt.Println(reqURL)
	err = cloud.SendFile(reqURL, file)
	fmt.Println(err)
	return err
}
