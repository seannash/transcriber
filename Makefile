.PHONY: all
all: lambda

.PHONY: lambda
lambda:
	ARTIFACTS_DIR=../../build/JobControllerProxy/JobControllerProxy make -C lambda/JobControllerProxy
	ARTIFACTS_DIR=../../build/StartTranscribeFromS3Event/StartTranscribeFromS3Event make -C lambda/StartTranscribeFromS3Event
	ARTIFACTS_DIR=../../build/TranscriberFinnish/TranscriberFinnish make -C lambda/TranscriberFinnish
	ARTIFACTS_DIR=../../build/SendEmail/SendEmail make -C lambda/SendEmail
	go build ./cmd/scribe

deploy-serverless:
	cd deployments/serverless/; sam deploy

.PHONY: clean
clean:
	rm -Rf build
	make -C lambda/JobController clean
	make -C lambda/StartTranscribe clean
	make -C lambda/TranscriberFinnish clean
