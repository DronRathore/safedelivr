![safeDelivr](http://i.imgur.com/ZGzd6kW.png)
-----------------------------------------------------------------------------------------------------------------

## Architecture
![Basic Architecture](http://i.imgur.com/YPTyw1X.png)

Safedelivr has the following core components
- User  : A github oAuth user
- Batch : An Email Batch
- Logs  : A status Log of each recipient
- Stats : A stats counter
- Worker: A queue consumer

## User
User is a basic entity of the safedelivr system. It has the following key properties.

|   User_Id     |      Email       |    Auth_Token     |     Api_Key       |
----------------|------------------|-------------------|-------------------|
|   Time UUID   |    varchar       |   Github Token    |    Unique Api_Key |

There isn't much to explain about the User entity as it is self explanatory.

## Batch
A Batch is a fundamental entity defining an email task, whenever the API is been hit, system generates a Batch_Id corresponding to the email delivery job.

A Batch has the following key parameters.

|   Batch_Id     |      User_Id               |    Subject              |     Options                                     |
-----------------|----------------------------|-------------------------|-------------------------------------------------|
|   Time UUID    |    The owner of this batch |   Subject Line of Email |    A map of options associated to this job      |

The _Options_ property of the Batch is a map which has the following keys

|     Key         |       Value                                     |
|-----------------|-------------------------------------------------|
|      _to_       |   A comma separated list of recipients         |
|     _from_      | Standard SMTP _from_ email address field       |
|     _body_      | HTML string                                     |
|    _reply_to_   |  Standard SMPTY _reply_to_ email address field  |

Whenever a client sends a batch creation task, system will acknowledge the request after performing necessary checks and will than save the job in the cassandra DB and publish a ```batch.batch-uuid``` key on the AMQP server.

A typical flow of batch creation is depicted below.
![Batch Creation](http://i.imgur.com/z11050c.png)
## Logs

A log is a bidirectional entity, it corresponds to an individual event/log for a given email address which has been dispatched by the provider. The logs are generated whenever the system receives an event corresponding to an email address via webhooks from the email provider. There are different types of events that are sent by the providers, the generic ones are the following.

|     Event Name     |         Meaning                                        |
|--------------------|--------------------------------------------------------|
|     processed      | Provider has dispatched the email.                     |
|     sent/delivered | Email has been successfully delivered                  |
|     dropped        | Mail has been dropped/bounced/hardfailed               |
|     delayed        | Mail service provider will retry sending the mail later|

From the above list, our system narrows down the events to ```success```, ```failure``` and ```queued```.

An email/log is said to be in __queued__ state if:
- Dispatched from our end, not yet received any status update.
- Failed by one/many providers and has been put back in queue for retrial with another provider.

An email/log is said to be in ___failed___ state if:
- Provider sends hardbounce/dropped/bounced event for that email.
- While retrying we exhausted out of the providers and don't have any options to retry with.

__Success__ state is self explanatory.
The main parameters of a Log doc are:

|     Log_Id          |           Batch_Id          |            User_Id          |        Email           |      State            |           Status                               |
----------------------|-----------------------------|-----------------------------|------------------------|-----------------------|------------------------------------------------|
|  Time UUID          |     Associated Batch_Id     |    Associated User_Id       |      Recipient Email   | queued/failed/success |      A map<> that holds the status of retrials |

The __status__ field is a boolean map which looks like this, it tells the system which provider we have retried with
```json
{
  "sendgrid": true,
  "mailgun":  false
}
```
## Stats

This is a meta doc which keeps track of counts for daily failures, success and queued emails in the system. Doc structure is pretty basic.

|    User_Id       |     Date        |      success        |      failure        |     queued         |
|------------------|-----------------|---------------------|---------------------|--------------------|
|   User Id        | Date of the stat|      counter        |       counter       |     counter        |

Overall Doc structure of complete system is shown below for reference.
![Doc Structure](http://i.imgur.com/getX1Mn.png)

## Consumers

Consumers are rabbitmq consumers listening on different queues, queues used by the system are categorized as:
- Generic Batch Processing consumer
- Individual Exclusive service provider based consumers
- Individual Exclusive Log retry consumers

### Generic Consumers

Generic consumers are associated to the batch processing part, they keeps on processing new enqueued batch that needs to be dispensed off.

Generic consumers listens for batches on the routing key ```batch.#```, where # corresponds to the batch's UUID.

Whenever a generic consumer fails to dispatch the mail batch request, it enqueues the packet into the exclusive consumer queue of a provider other than itself with an appended ```.retry._int_retry_count_``` routing key.
The appended ```retry``` field in routing key helps the exclusive queues in taking decision as to when to stop trying.

### Individual Exclusive Batch Consumers

These are service provider specific consumers, they will try to dispatch an email through the provider they are associated to e.g. Sendgrid, MailGun. They too follow the same method of retrial in case of failure i.e. Push into another service provider's queue.

### Individual Exclusive Log Consumer

These are also similar to individual batch consumers and consumes packets of ```log.provider_namespace.log-uuid``` signature. Whenever our system receives a delayed/failure response via webhooks by a service provider and we know we can retry sending mail to that particular recipient with another mail provider, the system will push that logid to another service provider's namespaced consumers.

Currently the signature of routing keys in the system are the following:

|      Routing Key                               |         Associated consumer          |
|------------------------------------------------|--------------------------------------|
|   _batch.00000000-0000-0000-0000-000000000000_   | Round Robin based (sendgrid/mailgun)  |
|   _mg.00000000-0000-0000-0000-000000000000_      |     __Mailgun__                          |
|   _sg.00000000-0000-0000-0000-000000000000_      |     __Sendgrid__                         |
|   _mg.00000000-0000-0000-0000-000000000000_.__retry.#num__|   __Mailgun__                       |
|   _sg.00000000-0000-0000-0000-000000000000_.__retry.#num__|   __Sendgrid__                      |
|   _log.mg.00000000-0000-0000-0000-000000000000_       |   __Mailgun__                       |
|   _log.sg.00000000-0000-0000-0000-000000000000_       |   __Sendgrid__                      |
|   _log.sg.00000000-0000-0000-0000-000000000000_.__retry.#num__|  __Sendgrid__                   |
|   _log.mg.00000000-0000-0000-0000-000000000000_.__retry.#num__|  __Mailgun__                    |

## Fail safe mechanism

Whenever the system encounters a failure, it is bound to retry until any one of the following condition is met:
- Successfully dispatched through one of the provider
- Ran out of service providers
- Ran out of number of retries

### Batch Lifecycle

A lifecycle of a Batch in fail safe mode is shown below
![Batch Lifecycle](http://i.imgur.com/vyGGuDe.png)

### Log Lifecycle
Lifecycle for a log in fail safe mode.
![Log Lifecycle](http://i.imgur.com/iWyhp6c.png)

## Adding Email Provider

Addition of a new Email provider is quite easy as long as the provider has webhook feedback mechanism, to add a new email provider you will need to add the following things:
- Add a Channel and append it to the [rabbit.EmailProvideres](https://github.com/DronRathore/safedelivr/blob/master/src/rabbit/setup.go#L47)
- Implement a consumer. Consumers are generically executed, have a look at the [sendgrid one](https://github.com/DronRathore/safedelivr/blob/master/src/worker/sendgrid.go#L26) to get an idea.
- Add a [webhook controller](https://github.com/DronRathore/safedelivr/blob/master/src/controller/webhook.go#L23-L203) for the same.
## Architecture flaws
- Current architecture doesn't keep state of consumer based failures, we can have a consumer based failure states so that they can be taken offline if any of them is failing rigorously
- Instead of immediately pushing for retry in case of a provider failure we can use [Dead Letter Exchange](http://yuserinterface.com/dev/2013/01/08/how-to-schedule-delay-messages-with-rabbitmq-using-a-dead-letter-exchange/) mechanism and add a delay in our retries, optionally we can make use of go context timeouts.
## Future Scope
 - An addition of a cron like consumer for delayed or unacknowledged entity will enhance the whole system's robustness.
 - Allow user defined webhooks to make it a completely service driven framework
 
MIT LICENSED
