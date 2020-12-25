NAME=$(shell basename $(shell pwd))

all: golang
	zip ${NAME}.zip ${NAME}

golang:
	go build .

clean:
	rm -f ${NAME} *zip
