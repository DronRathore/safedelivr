package main
import (
  "config"
  "redis"
  "cassandra"
  "router"
	express "github.com/DronRathore/goexpress"
)
 func main(){
  // read configuration
  config.Init()
  cassandra.Connect(config.Configuration.Cassandra.Ip, config.Configuration.Cassandra.Keyspace)
  // Initialise redis connection
  redis.Init()

  var Express = express.Express()
  router.SetRoutes(Express)
  Express.Start("8080")
 }