package main
import (
  "config"
  "redis"
  "cassandra"
  "rabbit"
  "router"
  "doggo"
	express "github.com/DronRathore/goexpress"
)
 func main(){
  // read configuration
  config.Init()
  // dial doggo
  doggo.DialDoggo()
  // connect amqp
  rabbit.Connect()
  // declare exchanges and queues
  rabbit.Setup()
  // connect to cassnadra
  cassandra.Connect(config.Configuration.Cassandra.Ip, config.Configuration.Cassandra.Keyspace)
  // setup DB initials
  cassandra.SetupDB()
  // Initialise redis connection
  redis.Init()

  var Express = express.Express()
  router.SetRoutes(Express)
  Express.Start(config.Configuration.Port)
  doggo.DoggoEvent("Server Started", "Port=" + config.Configuration.Port, false)
 }