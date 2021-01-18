package transcribe

import "fmt"

func MakeJobId(base string, num int64) string {
	return fmt.Sprintf("%s-%d", base, num)
}

type JobRecord struct {
	Job          string `json:"job"`
	User         string `json:"user"`
	JobStatus    string `json:"job_status"`
	SourceURI    string `json:"source_uri"`
	ResultBucket string `json:"result_bucket"`
	ResultKey    string `json:"result_key"`
}

type EmailMessage struct {
	To   string `json:"to"`
	Body string `json:"body"`
}
