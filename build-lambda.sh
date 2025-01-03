#!/usr/bin/env bash

cp go/config.yaml go/cmd/config.yaml
cd go/cmd

GOOS=linux GOARCH=amd64 go build -o bootstrap main.go

zip -j lambda_handler.zip bootstrap config.yaml

rm bootstrap config.yaml

mv lambda_handler.zip ../../deploy

cd ../../deploy

terraform apply
