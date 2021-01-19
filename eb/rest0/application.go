package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

func YourHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Gorilla!\n"))
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}
	f, _ := os.Create("/var/log/golang/golang-server.log")
	defer f.Close()
	log.SetOutput(f)

	r := mux.NewRouter()
	r.HandleFunc("/", YourHandler)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
