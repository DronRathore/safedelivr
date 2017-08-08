package cassandra

import (
  "fmt"
	"github.com/gocql/gocql"
)
// global session to query
var Session *gocql.Session

// Connect to a Cassandra node to a keyspace
func Connect(ip string, keyspace string) *gocql.Session {
  var err error
  cluster := gocql.NewCluster(ip)
  cluster.Keyspace = keyspace
  cluster.ProtoVersion = 4
  Session, err = cluster.CreateSession()
  if err != nil {
    fmt.Println("Cannot connect to the cassandra node/keyspace", ip, keyspace)
    panic(err)
  } else {
    return Session
  }
}

func Close(){
  Session.Close()
}
