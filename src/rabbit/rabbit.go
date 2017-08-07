package rabbit

import (
  "config"
  "fmt"
  "time"
  "github.com/streadway/amqp"
)

var AmqpConn *amqp.Connection
// const MSG_SIZE = 1024
// max retry to send a message
const MAX_MSG_RETRY = 3
type RabbitListener func(d amqp.Delivery)

// A custom Amqp Channel struct and functions
// that extends the functionality of Amqp
type Channel struct {
  Exchange string
  RoutingKey string
  QueueName string
  Ack bool
  Mandatory bool
  Immendiate bool
  channel *amqp.Channel
  Queue amqp.Queue
  Messages <- chan amqp.Delivery
}

func Connect() {
  var connString = "amqp://" + config.Configuration.Rabbit.ConnectString
  var err error
  AmqpConn, err = amqp.Dial(connString)
  if err != nil {
    fmt.Println("Cannot connect to RaabitMQ", err)
    panic(err)
  }
  fmt.Println("Connected to Rabbitmq")
}
// Create a new amqp Channel
func (c* Channel) NewChannel() error {
  channel, err := AmqpConn.Channel()
  if err != nil {
    fmt.Println(err)
    return err
  }
  c.channel = channel
  return nil
}
// Declare Exchange
func (c *Channel) DeclareExchange() error {
  err := c.channel.ExchangeDeclare(
    c.Exchange,
    "topic",
    true,
    false,
    false,
    false,
    nil)
  return err
}
// Set QOS option
func (c *Channel) SetQos(num int){
  c.channel.Qos(num, 0, false)
}
// Create a Queue and connect to exchange with defined route keys
func (c *Channel) DeclareQueue() bool {
  var err error
  c.Queue, err = c.channel.QueueDeclare(
    c.QueueName,
    false,
    false,
    false,
    false,
    nil)
  if err != nil {
    fmt.Println("Cannot declare a queue", err)
    return false
  }
  return true
}
// Bind the Queue to the Exchange
func (c *Channel) Bind() bool {
  err := c.channel.QueueBind(c.Queue.Name, c.RoutingKey, c.Exchange, false, nil)
  if err != nil {
    fmt.Println("Cannot bind queue to ", c.Exchange, err)
    return false
  }
  return true
}
/*
  Attaches a consumer listener on defined routing key and config
*/
func (c *Channel) Listen(listener RabbitListener) bool {
  var err error
  c.Messages, err = c.channel.Consume(
    c.Queue.Name,
    "",
    false,
    false,
    false,
    false,
    nil)
  if err != nil {
    fmt.Println("Cannot consume from ", c.Queue.Name, err)
    return false
  }
  go func(m <- chan amqp.Delivery){
    for message := range m {
      go listener(message)
    }
  }(c.Messages)
  return true
}
/*
  Publish on Queue
*/
func (c *Channel) Publish(routingKey string, data []byte) bool {
  var retryCount = 0
  Entry:
  msg := amqp.Publishing{
    DeliveryMode: amqp.Persistent,
    Timestamp:    time.Now(),
    ContentType:  "text/plain",
    Body:         []byte(""),
  }
  // make it a mandatory publish so we don't lose packets
  err := c.channel.Publish(c.Exchange, routingKey, true, false, msg)
  if err != nil {
    // todo: log error
    retryCount = retryCount + 1
    if retryCount < MAX_MSG_RETRY {
      goto Entry
    }
    return false
  }
  return true
}
func Close(){
  AmqpConn.Close()
}