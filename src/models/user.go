package models
import (
  "errors"
  "fmt"
  "time"
  "github.com/gocql/gocql"
  "cassandra"
)
type User struct {
  user_id gocql.UUID
  Email string
  Name string
  Api_Key string
  // Client_id string : todo: add a client id and let user have multi api_keys
  Location string
  User_Type int
  Auth_Token string
  Company string
  Created_at time.Time
}
// Queried User to User struct
func (u *User) Populate (data map[string]interface{}) *User {
  // Type safe injection
  if data["user_id"] != nil {
    u.user_id = data["user_id"].(gocql.UUID)
  }
  if data["email"] != nil {
    u.Email = data["email"].(string)
  }
  if data["name"] != nil {
    u.Name = data["name"].(string)
  }
  if data["api_key"] != nil {
    u.Api_Key = data["api_key"].(string)
  }
  if data["location"] != nil {
    u.Location = data["location"].(string)
  }
  if data["user_type"] != nil {
    u.User_Type = data["user_type"].(int)
  }
  if data["auth_token"] != nil {
    u.Auth_Token = data["auth_token"].(string)
  }
  if data["company"] != nil {
    u.Company = data["company"].(string)
  }
  if data["created_at"] != nil {
    u.Created_at = data["created_at"].(time.Time)
  }
  return u
}

func (u *User) GetId() string {
  return u.user_id.String()
}
// Returns UUID in orignal format
func (u *User) GetUUID() gocql.UUID {
  return u.user_id
}
// set UUID v1
func (u *User) New () *User {
  u.user_id = gocql.TimeUUID()
  return u
}

func (u *User) Update(data map[string]interface{}) (bool, error) {
  if u.Email != "" {
    return cassandra.Update("users", map[string]interface{}{"email": u.Email}, data)
  }
  return false, errors.New("Populate the user first")
}

// Check if a User exists, if then populate the model
func (u *User) Exists() (bool, error) {
  var options  map[string]interface{} = make(map[string]interface{})
  // todo: better map and make for searching docs
  if u.Email == "" && u.Api_Key != "" {
    options["api_key"] = u.Api_Key
  } else {
    options["email"] = u.Email
  }
  iterator := cassandra.Select("users", "*", options, 1)
  var data map[string]interface{} = make(map[string]interface{})
  iterator.MapScan(data)
  fmt.Println(data)
  if len(data) == 0 {
    // no row found
    return false, nil
  }
  // user found, populate it
  u.Populate(data)
  return true, nil
}

// Create a new User, check user.Exists() before trying to create a new one
func (u *User) Create(data map[string]interface{}) (bool, error) {
  // set a UUID
  data["user_id"] = gocql.TimeUUID()
  done, err := cassandra.Insert("users", data)
  if done == true {
    u.Populate(data)
    return true, err
  } else {
    return false, err
  }
}