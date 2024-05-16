
BINARY_NAME=jobrunner
GOOS:=$(shell go env GOOS)
GOARCH:=$(shell go env GOARCH)
BIN=bin/

build: 
	go build -o ${BIN}/jobrunner-${GOOS}  cmd/jobrunner/jobrunner.go
	go build -o ${BIN}/waitforfile-${GOOS} cmd/waitforfile/waitforfile.go 

linux:
	 $(MAKE) GOARCH=amd64 GOOS=linux build
win:
	 $(MAKE) GOARCH=amd64 GOOS=windows build 

clean:
	go clean
	rm -f ${BIN}/*
