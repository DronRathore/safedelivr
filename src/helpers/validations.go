package helpers

import (
  "regexp"
  "strings"
  request "github.com/DronRathore/goexpress/request"
)
var EmailRegexp *regexp.Regexp
const emailregex string = "^([[:alpha:]_.-]{1,})(@){1}([a-zA-Z0-9-]{1,})([.]){1}([a-z.]+)$"

var InitRegex = (func() bool {
  EmailRegexp = regexp.MustCompile(emailregex)
  return true
}())

func IsEmail(email string) bool {
  email = strings.ToLower(email)
  return EmailRegexp.MatchString(email)
}
/*
  Validates a batch request body
*/
func HasEssentials(req *request.Request) bool {
  if req.Body["from"] == nil || len(req.Body["from"]) == 0 || IsEmail(req.Body["from"][0]) == false {
    return false
  }
  if req.Body["to"] != nil {
    if len(req.Body["to"]) == 0 {
      return false
    }
    for _, email := range req.Body["to"] {
      if IsEmail(email) == false {
        return false
      }
    }
  }
  if req.Body["subject"] == nil || len(req.Body["subject"]) == 0 {
    return false
  }
  if req.Body["body"] == nil || len(req.Body["body"]) == 0 {
    return false
  }
  return true
}
