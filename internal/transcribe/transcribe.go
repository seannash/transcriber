package transcribe

import (
	"fmt"

	"example.com/transcribe/internal/types"
	"github.com/aws/aws-sdk-go/service/transcribeservice"
)

func MakeJobId(base string, num int64) string {
	return fmt.Sprintf("%s-%d", base, num)
}

func CallTranscribe(tranService *transcribeservice.TranscribeService, record types.JobRecord) error {

	mediaformat := "mp4"
	languagecode := "en-US"

	var media transcribeservice.Media
	media.MediaFileUri = &record.SourceURI

	params := transcribeservice.StartTranscriptionJobInput{
		TranscriptionJobName: &record.Job,
		Media:                &media,
		MediaFormat:          &mediaformat,
		LanguageCode:         &languagecode,
		OutputBucketName:     &record.ResultBucket,
		OutputKey:            &record.ResultKey,
	}
	fmt.Println(params)
	_, err := tranService.StartTranscriptionJob(&params)
	if err != nil {
		fmt.Println(err.Error())
	}
	return err
}
