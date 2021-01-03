.PHONY: all
all: lambda

.PHONY: lambda
lambda:
	ARTIFACTS_DIR=../../build/JobController/JobController make -C lambda/JobController
	ARTIFACTS_DIR=../../build/StartTranscribe/StartTranscribe make -C lambda/StartTranscribe
	ARTIFACTS_DIR=../../build/TranscriberFinnish/TranscriberFinnish make -C lambda/TranscriberFinnish
	go build ./cmd/scribe

deploy:
	sam deploy

.PHONY: clean
clean:
	rm -Rf build
	make -C lambda/JobController clean
	make -C lambda/StartTranscribe clean
	make -C lambda/TranscriberFinnish clean
