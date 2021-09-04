package main

type JobRecord struct {
	Job          string `json:"job"`
	User         string `json:"user"`
	JobStatus    string `json:"job_status"`
	SourceURI    string `json:"source_uri"`
	ResultBucket string `json:"result_bucket"`
	ResultKey    string `json:"result_key"`
}
