![travis](https://travis-ci.org/DronRathore/safedelivr.svg?branch=master)
![safeDelivr](http://i.imgur.com/ZGzd6kW.png)
# [SafeDelivr](https://safedelivr.com/)
If you are looking for the source of UI app of safedelivr, you can find it [here](https://github.com/DronRathore/safedelivr-ui)

An abstract documentation can also be found [here](https://safedelivr.com/docs).

Detailed Architecture documentation [is here](https://github.com/DronRathore/safedelivr/blob/master/architecture.md).

## Requirements
In order to use safedelivr, you will need the following:
- cassandra@3.7
- rabbitmq@3.6.9
- redis@4.0.1
- golang@1.8.3
- nodejs@8.0.1 (for UI server)
- datadog-agent aka statsd
- supervisord (optional)
- nginx (to proxy pass between frontend and backend, config is included with the code)

## Steps
Make sure you have installed the above required packages, after doing so run the below commands
```
cp ./application.yml.sample ./application.yml
./build.sh
```
This should compile the binaries and if you have supervisor than you can start the consumers as well as the app server by passing ```--deploy``` option to it.
```
./build.sh --deploy
```

## Troubleshoot
If ```cqlsh``` throws error of ```keyspace``` run the below command
```sh
cqlsh -e "CREATE KEYSPACE IF NOT EXISTS safedelivr WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}"
```
If ```cqlsh``` throws version error than you can force cqlsh to use a defined version by adding ```--cqlversion=version```
### Why Cassandra?
An Email system is generally a heavy transactional system with heavy DB writes, in case of a large scale architecture cassandra serves both as a good heavy write and high availaibility distributed storage.

### Why Rabbitmq?
Well I could have gone with Kafka too, but I was more familiar with Rabbitmq so chose the later one.

## Sending your first Email
In order to start sending emails you must first login and acquire your user_id and api_key from the Settings page. After that its just a form request away.
```sh
curl -X POST -H "Content-Type: application/x-www-form-urlencoded" --data 'from=me@me.com&to=you@you.com&subject=Holla&body=Hello&user_id=user_id&api_key=api_key' https://safedelivr.com/api/batch/
```
## Sending bulk email
To send a bulk email you just need to add the parameter ```is_bulk=true``` in your request.

## Adding Email Provider

Addition of a new Email provider is quite easy as long as the provider has webhook feedback mechanism, to add a new email provider you will need to add the following things:
- Add a Channel and append it to the [rabbit.EmailProvideres](https://github.com/DronRathore/safedelivr/blob/master/src/rabbit/setup.go#L47)
- Bind your listener to the Channel you have created. One individually and one for batch processing, add them [here](https://github.com/DronRathore/safedelivr/blob/master/worker.go#L31-L45).
- Implement a consumer. Consumers are generically executed, have a look at the [sendgrid one](https://github.com/DronRathore/safedelivr/blob/master/src/worker/sendgrid.go#L26) to get an idea.
- Add a [webhook controller](https://github.com/DronRathore/safedelivr/blob/master/src/controller/webhook.go#L23-L203) for the same.
## Failsafe decision helper

### [helpers.NextChannelToTry](https://github.com/DronRathore/safedelivr/blob/master/src/helpers/workers.go#L24)(log.Status) (worker)

This helper is used by the webhook controllers, it will return a new worker if any which can be retried.

Retrying within the consumers is handled automatically by the [generics file](https://github.com/DronRathore/safedelivr/blob/master/src/worker/generics.go#L120).
Future scopes and further granular details can be found in the [architecture documentation](https://github.com/DronRathore/safedelivr/blob/master/architecture.md).

## Things left for future addition
- User defined webhooks
- Allow attachments, and other meta Email fields.
