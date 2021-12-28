# Makefile for PNDPD

BINARY=pndpd
MODULES=
VERSION=`git describe --tags`
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.Build=${BUILD}"

build:
	go build -tags=${MODULES} -o bin/${BINARY} .

release:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags=${MODULES} ${LDFLAGS} -o bin/${BINARY}_linux_amd64.bin .

release-all:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags=${MODULES} ${LDFLAGS} -o bin/${BINARY}_linux_amd64.bin
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags=${MODULES} ${LDFLAGS} -o bin/${BINARY}_linux_arm64.bin
	CGO_ENABLED=0 GOOS=linux GOARCH=arm go build -tags=${MODULES} ${LDFLAGS} -o bin/${BINARY}_linux_arm.bin

clean:
	find bin/ -type f -delete
	if [ -d "bin/" ]; then rm -d bin/ ;fi