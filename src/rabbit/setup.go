package rabbit
import (
  // "config"
)
var BatchChannel Channel
var LogChannel Channel
var WebhookChannel Channel
var MailGunChannel Channel
var SendGridChannel Channel

var LogDB Channel

type WorkerList struct {
  Channel *Channel
  Namespace string
}

var EmailWorkers map[string]*WorkerList

func Setup(){
  // Declare Exchange & Queue for batch publish
  EmailWorkers = make(map[string]*WorkerList)

  BatchChannel = Channel{
    Exchange: "safedelivr",
    QueueName: "batches",
    RoutingKey: "batch.#",
  }
  BatchChannel.NewChannel()
  BatchChannel.DeclareExchange()
  BatchChannel.SetQos(1)
  BatchChannel.DeclareQueue()
  BatchChannel.Bind()

  // Dedicated MailGun Channel in case SG fails
  MailGunChannel = Channel{
    Exchange: "safedelivr",
    QueueName: "mg",
    RoutingKey: "mg.#",
  }
  MailGunChannel.NewChannel()
  MailGunChannel.DeclareExchange()
  MailGunChannel.SetQos(1)
  MailGunChannel.DeclareQueue()
  MailGunChannel.Bind()
  // add it to the workers list
  EmailWorkers["mailgun"] = &WorkerList{Channel: &MailGunChannel, Namespace: "mg."}
  
  // Dedicated SendGrid Channel in case MG fails
  SendGridChannel = Channel{
    Exchange: "safedelivr",
    QueueName: "sg",
    RoutingKey: "sg.#",
  }
  SendGridChannel.NewChannel()
  SendGridChannel.DeclareExchange()
  SendGridChannel.SetQos(1)
  SendGridChannel.DeclareQueue()
  SendGridChannel.Bind()
  // add it to the workers list
  EmailWorkers["sendgrid"] = &WorkerList{Channel: &SendGridChannel, Namespace: "sg."}
  
  // Declare Exchange & Queue for webhook processing
  // todo: Implement user-end webhook trigger from
  // service specific webhook data
  WebhookChannel = Channel{
    Exchange: "safedelivr",
    QueueName: "webhook",
    RoutingKey: "webhook.*",
  }
  WebhookChannel.NewChannel()
  WebhookChannel.DeclareExchange()
  WebhookChannel.SetQos(1)
  WebhookChannel.DeclareQueue()
  WebhookChannel.Bind()

  LogDB = Channel{
    Exchange: "safedelivr",
    QueueName: "logdb",
    RoutingKey: "logdb.*",
  }
  LogDB.NewChannel()
  LogDB.DeclareExchange()
  // LogDB.SetQos(1)
  LogDB.DeclareQueue()
  LogDB.Bind()
}