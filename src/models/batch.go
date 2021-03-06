// todo: generate models through scaffolds
package models
import (
  "errors"
  "time"
  "strconv"
  "github.com/gocql/gocql"
  "cassandra"
)
type Batch struct {
  Batch_id gocql.UUID
  User_id gocql.UUID
  // Client_key string // todo: make use of client_id
  subject string
  Status string
  reason string
  code string
  description string
  Options map[string]string
  Created_at time.Time
}
// Queried Batch to Batch struct
func (b *Batch) Populate (data map[string]interface{}) *Batch {
  // Type safe injections
  if data["user_id"] != nil {
    b.User_id = data["user_id"].(gocql.UUID)
  }
  if data["batch_id"] != nil {
    b.Batch_id = data["batch_id"].(gocql.UUID)
  }
  if data["subject"] != nil {
    b.subject = data["subject"].(string)
  }
  if data["status"] != nil {
    b.Status = data["status"].(string)
  }
  if data["options"] != nil {
    b.Options = data["options"].(map[string]string)
  }
  if data["created_at"] != nil {
    b.Created_at = data["created_at"].(time.Time)
  }
  if data["reason"] != nil {
    b.reason = data["reason"].(string)
  }
  if data["code"] != nil {
    b.reason = data["code"].(string)
  }
  if data["description"] != nil {
    b.description = data["description"].(string)
  }
  return b
}
// Return data in format of map for json response
func (b *Batch) GetMap() map[string]interface{} {
  var data = make(map[string]interface{})
  data["user_id"] = b.User_id
  data["created_at"] = b.Created_at
  data["options"] = b.Options
  data["status"] = b.Status
  data["subject"] = b.subject
  data["reason"] = b.reason
  data["code"] = b.code
  data["description"] = b.description
  return data
}

func (b *Batch) GetId() string {
  return b.Batch_id.String()
}
// set UUID v1
func NewBatch () *Batch {
  return &Batch{}
}

func (b *Batch) Update(data map[string]interface{}) (bool, error) {
  var empty gocql.UUID
  if b.Batch_id != empty {
    return cassandra.Update("batches", map[string]interface{}{"batch_id": b.Batch_id}, data)
  }
  return false, errors.New("Populate the batch first")
}

// Check if a Batch exists, if then populate the model
func (b *Batch) Exists() (bool, error) {
  var empty gocql.UUID
  var emptyTime time.Time
  //todo: better filtering options
  if b.Batch_id == empty {
    return false, errors.New("Need batch id to process")
  }
  var options map[string]interface{} = make(map[string]interface{})
  // query only through batch id
  if b.User_id != empty {
    options["user_id"] = b.User_id
  } 
  if b.Batch_id != empty {
    // query through all params
    options["batch_id"] = b.Batch_id
  }
  if b.Created_at != emptyTime {
    options["created_at"] = b.Created_at
  }
  iterator := cassandra.Select("batches", "*", options, 1)
  var data map[string]interface{} = make(map[string]interface{})
  iterator.MapScan(data)
  if len(data) == 0 {
    // no row found
    return false, nil
  }
  // user found, populate it
  b.Populate(data)
  return true, nil
}
/*
  Retrieves batches associated to a user
*/
func FetchBatches(user_id gocql.UUID, limit int) *[]map[string]interface{} {
  var query = "SELECT * from batches where user_id=" + user_id.String()
  query = query + " LIMIT " + strconv.Itoa(limit) + " ALLOW FILTERING"
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
// Create a new User, check user.Exists() before trying to create a new one
func (b *Batch) Create(data map[string]interface{}) (bool, error) {
  // set a UUID
  data["batch_id"] = gocql.TimeUUID()
  done, err := cassandra.Insert("batches", data)
  if done == true {
    b.Populate(data)
    return true, err
  } else {
    return false, err
  }
}