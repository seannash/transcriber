.PHONY: lambda
lambda:
	make -C lambda/JobController
	make -C lambda/StartTranscribe
	make -C lambda/TranscriberFinnish
	go build ./cmd/scribe


.PHONY: lambda
clean:
	make -C lambda/JobController clean
	make -C lambda/StartTranscribe clean
	make -C lambda/TranscriberFinnish clean
	rm -f scribe
