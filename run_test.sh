#!/bin/sh
# Currently facing problem with direct go -test
# hence this file
GOPATH=`pwd` go get
GOPATH=`pwd` go test ./src/helpers/validations.go ./src/helpers/validations_test.go
GOPATH=`pwd` go test ./src/helpers/workers.go ./src/helpers/workers_test.go
GOPATH=`pwd` go test ./src/helpers/webhook.go ./src/helpers/webhook_test.go