#!/bin/sh
export GOPATH=`pwd`
hasSupervisor=(which supervisord)

go get
go build worker.go
go build app.go
mkdir -p ./tmp

build=$1
if [ "$build" == "--deploy" ]; then
	if [ "hasSupervisor" == "" ]; then
		echo "Supervisor not found, exiting"
		exit
	fi
	echo Deploying..
	if [ ! -f ./tmp/supervisor.sock ]; then
		supervisord -c supervisord.conf
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
fi
