LAMBDAS=lambda/*

.PHONY: lambda
lambda:
	make -C lambda/JobController
	make -C lambda/StartTranscribe
	make -C lambda/TranscriberFinnish


.PHONY: lambda
clean:
	make -C lambda/JobController clean
	make -C lambda/StartTranscribe clean
	make -C lambda/TranscriberFinnish clean

