
BINARY_NAME=jobrunner
GOOS:=$(shell go env GOOS)
GOARCH:=$(shell go env GOARCH)
BIN=bin/

build: 
	go build -o ${BIN}/jobrunner${SFX}  cmd/jobrunner/jobrunner.go
	go build -o ${BIN}/waitforfile${SFX} cmd/waitforfile/waitforfile.go 

linux:
	 $(MAKE) GOARCH=amd64 GOOS=linux SFX="-linux" build
win:
	 $(MAKE) GOARCH=amd64 GOOS=windows SFX=".exe" build 

clean:
	go clean
	rm -f ${BIN}/*
