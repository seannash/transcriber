package transcribe

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"
)

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

func PostRequest(url string, token string) ([]byte, error) {
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		log.Fatal("Error reading request. ", err)
	}

	req.Header.Set("Authorization", token)

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

func SendFile(url string, filename string) error {

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	bs, err := ioutil.ReadFile(filename)

	//body := new(bytes.Buffer)

	request, _ := http.NewRequest("PUT", url, strings.NewReader(string(bs)))
	client := &http.Client{}

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
	_, err = io.Copy(out, resp.Body)
	return err
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
