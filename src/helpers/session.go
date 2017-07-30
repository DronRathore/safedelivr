package helpers
import (
  "models"
  "redis"
  "encoding/json"
  "errors"
)

func SaveSession(sessionId string, data *models.GithubUser) bool {
  dataToByteArray, err := json.Marshal(data)
  if err != nil {
    return false
  }
  err = redis.Redis.Set(sessionId, string(dataToByteArray))
  if err != nil {
    return false
  }
  return true
}

func GetSession(sessionId string) (*models.UserRedis, error) {
  var sessionData = redis.Redis.Get(sessionId)
  if sessionData != "" {
    var User = &models.UserRedis{}
    err := json.Unmarshal([]byte(sessionData), User)
    return User, err
  } else {
    return nil, errors.New("Cannot get data from redis")
  }
}