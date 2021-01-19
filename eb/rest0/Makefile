all:
	mkdir -p deployment
	go build -o deployment/bin/application application.go
	cd deployment; zip -r ../deployment.zip *

