#!/bin/sh
export GOPATH=`pwd`
go get
go build worker.go
go build app.go
mkdir -p ./tmp
if [ ! -f ./tmp/supervisor.sock ]; then
	supervisord
	supervisorctl stop worker:*
	supervisorctl start worker:*
	supervisorctl stop go:*
	supervisorctl start go:*
	supervisorctl status
else
	supervisorctl stop worker:*
	supervisorctl start worker:*
	supervisorctl stop go:*
	supervisorctl start go:*
fi