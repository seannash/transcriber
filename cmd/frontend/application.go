package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/lestrrat-go/jwx/jwk"

	"github.com/gorilla/mux"
	"github.com/urfave/negroni"
	//"github.com/dgrijalva/jwt-go"
	//"github.com/dgrijalva/jwt-go"
	//"github.com/gorilla/mux"
	//"github.com/lestrrat-go/jwx/jwk"
	//"github.com/urfave/negroni"
	//jwt "github.com/vladimiroff/jwt-go"
)

var (
	g_session       *session.Session
	g_db            dynamodbiface.DynamoDBAPI
	g_s3            s3iface.S3API
	g_projectBucket string
	g_projectTable  string
)

func main() {

	g_session, _ = session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)
	g_db = dynamodb.New(g_session)
	g_s3 = s3.New(g_session)
	g_projectBucket = os.Getenv("PROJECT_BUCKET")
	fmt.Println("Project Bucket: ", g_projectBucket)
	g_projectTable = os.Getenv("TABLE_NAME")
	fmt.Println("Project Table: ", g_projectTable)

	r := mux.NewRouter()
	r.HandleFunc("/ping", PingHandler)
	//sr := r.PathPrefix("/transcribe").Subrouter()
	r.HandleFunc("/transcribe/{user}/job", ListJobsHandler)
	r.HandleFunc("/transcribe/{user}/job/{id}", GetJobUriHandler)
	r.HandleFunc("/transcribe/{user}/upload/{id}", GetUploadUriHandler)

	keysUrl := "https://cognito-idp.us-east-1.amazonaws.com/us-east-1_gqhcPGBgB/.well-known/jwks.json"
	//ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	//jwks := jwk.NewAutoRefresh(ctx)
	keySet, _ := jwk.Fetch(keysUrl)
	fmt.Println(keySet)
	/*
		mw := jwtmiddleware.New(jwtmiddleware.Options{
			ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
				fmt.Println("Hi")
				//if _, ok := token.Method.(*jwt.SigningMethodRS256); !ok {
				//	return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
				//}
				kid, ok := token.Header["kid"].(string)
				if !ok {
					return nil, errors.New("kid header not found")
				}
				keys := keySet.LookupKeyID(kid)
				if len(keys) == 0 {
					return nil, fmt.Errorf("key %v not found", kid)
				}
				var raw interface{}
				err := keys[0].Raw(&raw)
				fmt.Println("err=", err)
				fmt.Println("raw=", raw)
				return raw, err

			},
			SigningMethod: jwt.SigningMethodRS256,
		})
	*/
	//an := negroni.New(negroni.HandlerFunc(mw.HandlerWithNext), negroni.Wrap(r))
	//r.PathPrefix("/").Handler(an)
	//r.PathPrefix("/transcribe").Handler(negroni.New(
	//	negroni.HandlerFunc(mw.HandlerWithNext),
	//	negroni.Wrap(sr),
	//))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	n := negroni.Classic()
	n.UseHandler(r)
	n.Run(":" + port)
}

func PingHandler(w http.ResponseWriter, r *http.Request) {
	WriteResponse(w, "", nil)
}

func ListJobsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Hi 2")
	// Get Path Variables
	vars := mux.Vars(r)
	user := vars["user"]
	// Perform Operation
	body, err := ListJobs(user, g_projectTable, g_db)
	// Write Response
	WriteResponse(w, body, err)
}

func ListJobs(user string, table string, dynaClient dynamodbiface.DynamoDBAPI) (*[]JobRecord, error) {
	params := &dynamodb.QueryInput{
		TableName:              aws.String(table),
		IndexName:              aws.String("user-index"),
		KeyConditionExpression: aws.String("#user = :user"),
		ExpressionAttributeNames: map[string]*string{
			"#user": aws.String("user"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":user": {
				S: aws.String(user),
			},
		},
	}

	resp, err := dynaClient.Query(params)
	if err != nil {
		fmt.Printf("ERROAR: %v\n", err.Error())
		return nil, err
	}

	fmt.Println(resp)

	items := new([]JobRecord)
	if resp.Items != nil {
		err = dynamodbattribute.UnmarshalListOfMaps(resp.Items, &items)
	}

	return items, nil
}

func GetJobUriHandler(w http.ResponseWriter, r *http.Request) {
	// Get Path Variables
	vars := mux.Vars(r)
	//user := vars["user"] // Should check permission
	id := vars["id"]
	// Perform Operation
	body, err := GetJobLocation(g_db, g_s3, g_projectTable, id)
	// Write Response
	WriteResponse(w, body, err)
}

func GetJobLocation(DB dynamodbiface.DynamoDBAPI, S3 s3iface.S3API, table string, id string) (string, error) {
	result, err := GetJob(id, table, DB)
	if err != nil {
		return "", err
	}
	resultBucket := result.ResultBucket
	resultKey := result.ResultKey
	return MakeSignedURI(S3, resultBucket, resultKey)
}

func GetJob(job string, tableName string, dynaClient dynamodbiface.DynamoDBAPI) (*JobRecord, error) {
	fmt.Println("GetJob")
	result, err := dynaClient.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"job": {
				S: aws.String(job),
			},
		},
	})
	item := new(JobRecord)
	if err != nil {
		fmt.Println(result)
		return item, errors.New("failed")
	}
	err = dynamodbattribute.UnmarshalMap(result.Item, item)
	if err != nil {
		return nil, errors.New("ErrorFailedToUnmarshalRecord")
	}
	return item, nil
}

func MakeSignedURI(s3Service s3iface.S3API, bucket string, key string) (string, error) {

	reqo, out := s3Service.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if out != nil {
		log.Println("Failed to create request", out)
	}
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
	fmt.Println("Here ", uri, err)
	if err != nil {
		log.Println("Failed to sign request", err)
	}

	return uri, err

}

func GetUploadUriHandler(w http.ResponseWriter, r *http.Request) {
	// Get Path Variables
	fmt.Println("GetUploadIUriHandler")
	vars := mux.Vars(r)
	user := vars["user"]
	id := vars["id"]
	fmt.Println("User: ", user)
	fmt.Println("id: ", id)
	// Perform Operation
	body, err := GetUploadUri(g_s3, g_projectBucket, user, id)
	// Write Response
	fmt.Println("Body", body, err)
	WriteResponse(w, body, err)
}

func GetUploadUri(S3 s3iface.S3API, bucket string, user string, id string) (string, error) {
	reqo, ou := S3.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("users/" + user + "/" + id),
	})
	fmt.Println(reqo)
	fmt.Println("OU: ", ou)
	urlStr, err := reqo.Presign(15 * time.Minute)
	fmt.Println("Got URL: ", urlStr, err)
	return urlStr, err
}

func WriteResponse(w http.ResponseWriter, body interface{}, err error) {
	jbody, err := json.Marshal(body)
	if err != nil {
		w.WriteHeader(http.StatusOK)
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, string(jbody))
}

type JobRecord struct {
	Job          string `json:"job"`
	User         string `json:"user"`
	JobStatus    string `json:"job_status"`
	SourceURI    string `json:"source_uri"`
	ResultBucket string `json:"result_bucket"`
	ResultKey    string `json:"result_key"`
}
