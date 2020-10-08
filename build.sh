#!/bin/sh
go build -o ddns
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ddns-amd64
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o ddns-arm64
GOOS=linux GOARCH=ppc64le go build -ldflags="-s -w" -o ddns-ppc64le