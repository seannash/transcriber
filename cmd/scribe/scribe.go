package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
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

	transcribeCommand := flag.NewFlagSet("transcribe", flag.ExitOnError)
	arg.fileName = transcribeCommand.String("filename", "123", "The file to be uploaded")
	listCommand := flag.NewFlagSet("list", flag.ExitOnError)

	getCommand := flag.NewFlagSet("get", flag.ExitOnError)
	arg.job = getCommand.String("job", "", "The job id")

	switch os.Args[1] {
	case "transcribe":
		transcribeCommand.Parse(os.Args[2:])
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
	if transcribeCommand.Parsed() {
		fmt.Println("Here")
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
	fmt.Println(params)
	cip := cognitoidentityprovider.New(sess)
	authResp, err := cip.InitiateAuth(params)
	fmt.Println(authResp, err)
	return authResp, err
}

func ListJobs(arg args, config Config) error {
	sess, _ := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{Credentials: nil},
	})
	lparams := LoginParams{
		ApiKey:   config.ApiKey,
		UserName: config.UserName,
		Password: config.Password,
	}
	authRequestOutput, _ := Login(sess, lparams)
	fmt.Println(authRequestOutput)
	token := *authRequestOutput.AuthenticationResult.IdToken
	url := config.Api + "/transcribe/" + config.UserName + "/job"
	data, err := GetRequest(url, token)
	fmt.Println("Response: ", string(data), err)
	return err
}

func GetJob(arg args, config Config) error {
	sess, _ := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{Credentials: nil},
	})
	lparams := LoginParams{
		ApiKey:   config.ApiKey,
		UserName: config.UserName,
		Password: config.Password,
	}
	authRequestOutput, _ := Login(sess, lparams)
	token := *authRequestOutput.AuthenticationResult.IdToken
	uri := config.Api + "/transcribe/" + config.UserName + "/job/" + *arg.job
	fmt.Println(uri)
	data, err := GetRequest(uri, token)
	fmt.Println(string(data), err)
	var url string
	json.Unmarshal(data, &url)
	json, _ := GetString(url)
	fmt.Println(json)
	return err
}

func GetRequest(url string, token string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal("Error reading request. ", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	//req.Header.Set("Authorization", token)
	client := &http.Client{Timeout: time.Second * 10}
	requestDump, err := httputil.DumpRequest(req, true)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(requestDump))
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

func GetString(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	buf := new(strings.Builder)
	_, err = io.Copy(buf, resp.Body)
	return buf.String(), err
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
	sess, _ := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{Credentials: nil},
	})
	lparams := LoginParams{
		ApiKey:   config.ApiKey,
		UserName: config.UserName,
		Password: config.Password,
	}
	authRequestOutput, _ := Login(sess, lparams)
	token := *authRequestOutput.AuthenticationResult.IdToken
	user := config.UserName
	file := *arg.fileName
	fmt.Println("File ", file, " <-")
	api := config.Api
	url := api + "/transcribe/" + user + "/upload/" + file
	data, _ := GetRequest(url, token)
	fmt.Println(string(data)) // Check 404
	var reqURL string
	json.Unmarshal(data, &reqURL)
	err := SendFile(reqURL, file)
	return err
}

func SendFile(url string, filename string) error {

	fmt.Println("Sending!!! ", url, filename)
	fmt.Println(url)
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("fuck")
		return err
	}
	defer file.Close()
	bs, _ := ioutil.ReadFile(filename)

	//body := new(bytes.Buffer)

	request, _ := http.NewRequest("PUT", url, strings.NewReader(string(bs)))
	client := &http.Client{}
	fmt.Println(request)
	resp, err := client.Do(request)
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
