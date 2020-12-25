package types

type JobRecord struct {
	Job       string `json:"job"`
	User      string `json:"user"`
	JobStatus string `json:"job_status"`
	SourceUrl string `json:"source_uri"`
}
