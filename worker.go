package main

import (
  "config"
  "worker"
  "cassandra"
  "rabbit"
  "doggo"
  "fmt"
)

var exit chan bool
func main() {
  // read configuration
  config.Init()
  // connect amqp
  rabbit.Connect()
  // declare exchanges and queues
  rabbit.Setup()
  doggo.DialDoggo()
  // connect to cassnadra
  cassandra.Connect(config.Configuration.Cassandra.Ip, config.Configuration.Cassandra.Keyspace)
  worker.InitSg()
  worker.InitMg()
  // attach listeners to the queue
  fmt.Println("Starting consumers...")

  ///////////////// Global Queue ////////////////
  var workerCounts = config.Configuration.Consumers
  // sendgrid
  for i := 1; i <= workerCounts; i = i + 1 {
    rabbit.BatchChannel.Listen(worker.SGListener)  
    fmt.Println(i, " Sendgrid Consumer started")
    fmt.Println(i, " Mail Gun Consumer started")
    rabbit.BatchChannel.Listen(worker.MGListener)
  }

  ////////////// Fail Recovery Queues ///////////////
  workerCounts = config.Configuration.Consumers
  for i := 1; i <= workerCounts; i = i + 1 {
    rabbit.SendGridChannel.Listen(worker.SGListener)
    fmt.Println(i, " Sendgrid Individual Consumer started")
    fmt.Println(i, " Mail Gun Individual Consumer started")
    rabbit.MailGunChannel.Listen(worker.MGListener)  
  }
  doggo.DoggoEvent("Doggo consumers ready", "success" , false)
  // A quick hack to put the worker in infinte loop
  <- exit
}