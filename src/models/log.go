// todo: generate models through scaffolds
package models
import (
  "errors"
  "time"
  "github.com/gocql/gocql"
  "strconv"
  "cassandra"
)
type Log struct {
  Log_id gocql.UUID
  Batch_id gocql.UUID
  User_id gocql.UUID
  // Client_key string // todo: make use of client_id
  Email string
  State string
  Status map[string]bool // a map of service providers state
  LastUpdate time.Time
  Created_at time.Time
}
// Queried Batch to Batch struct
func (l *Log) Populate (data map[string]interface{}) *Log {
  // Type safe injections
  if data["log_id"] != nil {
    l.Log_id = data["log_id"].(gocql.UUID)
  }
  if data["user_id"] != nil {
    l.User_id = data["user_id"].(gocql.UUID)
  }
  if data["batch_id"] != nil {
    l.Batch_id = data["batch_id"].(gocql.UUID)
  }
  if data["email"] != nil {
    l.Email = data["email"].(string)
  }
  if data["state"] != nil {
    l.State = data["state"].(string)
  }
  if data["status"] != nil {
    l.Status = data["status"].(map[string]bool)
  }
  if data["created_at"] != nil {
    l.Created_at = data["created_at"].(time.Time)
  }
  if data["last_update"] != nil {
    l.LastUpdate = data["last_update"].(time.Time)
  }
  return l
}
// Return data in format of map for json response
func (l *Log) GetMap() map[string]interface{} {
  var data = make(map[string]interface{})
  data["user_id"] = l.User_id
  data["log_id"] = l.Log_id
  data["batch_id"] = l.Batch_id
  data["created_at"] = l.Created_at
  data["status"] = l.Status
  data["state"] = l.State
  data["email"] = l.Email
  data["last_update"] = l.LastUpdate
  return data
}

func (l *Log) GetId() string {
  return l.Log_id.String()
}
// set UUID v1
func NewLog () *Log {
  return &Log{}
}

func (l *Log) Update(data map[string]interface{}) (bool, error) {
  var empty gocql.UUID
  if l.Log_id != empty {
    return cassandra.Update("logs", map[string]interface{}{"log_id": l.Log_id}, data)
  }
  return false, errors.New("Populate the log first")
}

// Check if a Batch exists, if then populate the model
func (l *Log) Exists() (bool, error) {
  var empty gocql.UUID
  //todo: better filtering options
  // only queries on the basis of api_key
  if l.Log_id == empty {
    return false, errors.New("Need log id to process")
  }
  var options map[string]interface{} = make(map[string]interface{})
  options["log_id"] = l.Log_id
  // query only through batch id
  if l.Batch_id != empty {
    options["batch_id"] = l.Batch_id
  }
  if l.Email != "" {
    options["email"] = l.Email
  } 
  iterator := cassandra.Select("logs", "*", options, 1)
  var data map[string]interface{} = make(map[string]interface{})
  iterator.MapScan(data)
  if len(data) == 0 {
    // no row found
    return false, nil
  }
  // user found, populate it
  l.Populate(data)
  return true, nil
}
// Fetch array of logs
func FetchLogs(batch_id gocql.UUID, page gocql.UUID, order_by string, limit int) (*[]map[string]interface{}) {
  var emptyId gocql.UUID
  if batch_id == emptyId {
    return nil
  }
  var query = "Select * from logs where batch_id=" + batch_id.String()
  limitStr := strconv.Itoa(limit)
  query = query + " LIMIT " + limitStr
  iterator := cassandra.Session.Query(query).Consistency(gocql.One).Iter()
  var logs = make([]map[string]interface{}, 0)
  for {
    var data = make(map[string]interface{})
    if !iterator.MapScan(data) {
      break;
    }
    logs = append(logs, data)
  }
  return &logs
}
// Creates a new Log doc
func (l *Log) Create(data map[string]interface{}) (bool, error) {
  // set a UUID
  data["log_id"] = gocql.TimeUUID()
  done, err := cassandra.Insert("logs", data)
  if done == true {
    l.Populate(data)
    return true, err
  } else {
    return false, err
  }
}