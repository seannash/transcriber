package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"example.com/transcribe/internal/transcribe"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/gorilla/mux"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/urfave/negroni"

	//"github.com/dgrijalva/jwt-go"

	jwtmiddleware "github.com/auth0/go-jwt-middleware"
	"github.com/form3tech-oss/jwt-go"
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
	g_projectTable = os.Getenv("PROJECT_TABLE")
	fmt.Println("Project Table: ", g_projectTable)

	r := mux.NewRouter()
	r.HandleFunc("/transcribe/{user}/job", ListJobsHandler)
	r.HandleFunc("/transcribe/{user}/job/{id}", GetJobUriHandler)
	r.HandleFunc("/transcribe/{user}/upload", GetUploadUriHandler)

	keysUrl := "https://cognito-idp.us-east-1.amazonaws.com/us-east-1_EkYgDQt5n/.well-known/jwks.json"
	//ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	//jwks := jwk.NewAutoRefresh(ctx)
	keySet, err := jwk.Fetch(keysUrl)
	fmt.Println(err)
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

	an := negroni.New(negroni.HandlerFunc(mw.HandlerWithNext), negroni.Wrap(r))
	//r.PathPrefix("/").Handler(an)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}
	n := negroni.Classic()
	n.UseHandler(an)
	n.Run(":" + port)
}

func ListJobsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Hi 2")
	// Get Path Variables
	vars := mux.Vars(r)
	user := vars["user"]
	// Perform Operation
	body, err := transcribe.ListJobs(user, g_projectTable, g_db)
	// Write Response
	WriteResponse(w, body, err)
}

func GetJobUriHandler(w http.ResponseWriter, r *http.Request) {
	// Get Path Variables
	vars := mux.Vars(r)
	//user := vars["user"] // Should check permission
	id := vars["id"]
	// Perform Operation
	body, err := transcribe.GetJobLocation(g_db, g_s3, g_projectTable, id)
	// Write Response
	WriteResponse(w, body, err)
}

func GetUploadUriHandler(w http.ResponseWriter, r *http.Request) {
	// Get Path Variables
	vars := mux.Vars(r)
	user := vars["user"]
	id := vars["id"]
	// Perform Operation
	body, err := transcribe.GetUploadUri(g_s3, g_projectBucket, user, id)
	// Write Response
	WriteResponse(w, body, err)
}

func WriteResponse(w http.ResponseWriter, body interface{}, err error) {
	jbody, err := json.Marshal(body)
	if err != nil {
		w.WriteHeader(http.StatusOK)
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, string(jbody))
}
