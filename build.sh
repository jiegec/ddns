#!/bin/sh
go build -o ddns
export CGO_ENABLED=0
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o ddns-windows-amd64
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o ddns-darwin-amd64
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ddns-linux-amd64
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o ddns-linux-arm64
GOOS=linux GOARCH=ppc64le go build -ldflags="-s -w" -o ddns-linux-ppc64le
GOOS=linux GOARCH=mipsle go build -ldflags="-s -w" -o ddns-linux-mipsle
upx ddns-*
