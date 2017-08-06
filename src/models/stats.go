package models
import (
  "time"
  "strings"
  "github.com/gocql/gocql"
  "cassandra"
)

type Stat struct {
  Failed int64
  Success int64
  Queued int64
  User_Id gocql.UUID
  Date time.Time
}

func NewStat(user_id gocql.UUID) *Stat {
  t := time.Now()
  rounded := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
  return &Stat{Date: rounded, User_Id: user_id}
}
/*
  Returns stats of a week
*/
func GetWeekStats(user_id gocql.UUID) (*[]map[string]interface{}) {
  t := time.Now()
  var lastDay = t.Day() + 1
  rounded := time.Date(t.Year(), t.Month(), lastDay - 7, 0, 0, 0, 0, time.UTC)
  timeParts := strings.Split(rounded.String(), "+0000")
  timeStr := timeParts[0] + timeParts[1]
  var query = "Select * from stats where user_id=" + user_id.String()
  query = query + " AND date >= '" + timeStr + "'"
  iterator := cassandra.Session.Query(query).Consistency(gocql.One).Iter()
  var stats = make([]map[string]interface{}, 0)
  for {
    var data = make(map[string]interface{})
    if !iterator.MapScan(data) {
      break;
    }
    stats = append(stats, data)
  }
  return &stats
}

func (s *Stat) Populate(data map[string]interface{}){
  if data["failed"] != nil {
    s.Failed = data["failed"].(int64)
  }
  if data["success"] != nil {
    s.Success = data["success"].(int64)
  }
  if data["queued"] != nil {
    s.Queued = data["queued"].(int64)
  }
}

func (s *Stat) Exists() bool {
  var options = make(map[string]interface{})
  options["date"] = s.Date
  options["user_id"] = s.User_Id
  iterator := cassandra.Select("stats", "*", options, 1)
  var data map[string]interface{} = make(map[string]interface{})
  iterator.MapScan(data)
  if len(data) == 0 {
    // no row found
    return false
  }
  // stats found, populate it
  s.Populate(data)
  return true
}

func (s *Stat) Update(options []string) bool {
  var query = "Update stats SET "
  for _, key := range options {
    query = query + key + "=" + key + "+1,"
  }
  query = query[:len(query)-1]
  query = query + " WHERE user_id=?"
  query = query + " AND date=?"
  err := cassandra.Session.Query(query, s.User_Id, s.Date).Exec()
  if err != nil {
    return false
  } else {
    s.Exists()
  }
  return true
}