package helpers
import (
  "config"
  "io"
  "mime/multipart"
  "strings"
  "models"
  "crypto/sha256"
  "crypto/hmac"
  "encoding/hex"
  "github.com/gocql/gocql"
  request "github.com/DronRathore/goexpress/request"
)
const MAX_FORM_SIZE = 1024*1024*1024

func ReadMultiPartForm(boundary string, body io.Reader) *multipart.Form {
  reader := multipart.NewReader(body, boundary)
  form,err := reader.ReadForm(MAX_FORM_SIZE)
  if err != nil {
    return nil
  } else {
    return form
  }
}
/*
  Updates stats for the day
*/
func UpdateStats(user_id gocql.UUID, key string){
  stat := models.NewStat(user_id)
  stat.Update([]string{key})
}
func IsMultipart(header string, boundary *string) bool {
  parts := strings.Split(header, ";")
  if len(parts) == 2 {
    parts := strings.Split(parts[1], "=")
    if len(parts) == 2 && strings.TrimSpace(parts[0]) == "boundary" {
      *boundary = parts[1]
      return true
    }
  }
  return false
}

func IsValidMGRequest(req *request.Request) bool {
  if req.Body["timestamp"] == nil || req.Body["token"] == nil || req.Body["signature"] == nil {
    return false
  }
  var candidate = req.Body["timestamp"][0] + req.Body["token"][0]
  var hashKey = []byte(config.Configuration.MailGun.Key)
  var hash = hmac.New(sha256.New, hashKey)
  hash.Write([]byte(candidate))
  return hex.EncodeToString(hash.Sum(nil)) == req.Body["signature"][0]
}