package transcribe

import (
	"fmt"
	"time"

	"example.com/transcribe/internal/types"
	"github.com/aws/aws-sdk-go/service/transcribeservice"
)

func CallTranscribe(tranService *transcribeservice.TranscribeService, record types.JobRecord) error {
	now := time.Now()
	sec := now.Unix()
	jobname := fmt.Sprintf("1-%d", sec)

	mediaformat := "mp4"
	languagecode := "en-US"

	var media transcribeservice.Media
	media.MediaFileUri = &record.SourceUrl

	params := transcribeservice.StartTranscriptionJobInput{
		TranscriptionJobName: &jobname,
		Media:                &media,
		MediaFormat:          &mediaformat,
		LanguageCode:         &languagecode,
	}
	_, err := tranService.StartTranscriptionJob(&params)
	if err != nil {
		fmt.Println(err.Error())
	}
	return err
}
