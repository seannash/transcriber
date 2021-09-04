.PHONY: all
all: lambda

.PHONY: lambda
lambda:
	mkdir -p build/StartTranscribeFromS3EventLambda
	mkdir -p build/TranscriberFinishLambda
	mkdir -p build/SendEmailLambda
	go build -o build/StartTranscribeFromS3EventLambda/StartTranscribeFromS3EventLambda cmd/StartTranscribeFromS3EventLambda/main.go
	go build -o build/TranscriberFinishLambda/TranscriberFinishLambda cmd/TranscriberFinishLambda/main.go
	go build -o build/SendEmailLambda/SendEmailLambda cmd/SendEmailLambda/main.go
	go build ./cmd/scribe

.PHONY: clean
clean:
	rm -Rf build
	
