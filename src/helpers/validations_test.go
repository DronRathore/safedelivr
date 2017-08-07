package helpers

import(
	"testing"
  request "github.com/DronRathore/goexpress/request"
)

func TestInitRegex(t *testing.T){
  if InitRegex != true {
    t.Log("Failed to compile email regex")
    t.Fail()
  }
}

func TestIsEmail(t *testing.T){
  t.Log("Testing helpers.IsEmail")
  var samples = map[string]bool{
    "sample@sample.com" : true,
    "sample-@sample.com": true,
    "sample-sample@sample.com": true,
    "sample.sample@sample.co.mn": true,
    "sample.sample@sample": false,
    "": false,
  }
  for email, shouldMatch := range samples {
    if IsEmail(email) != shouldMatch {
      t.Log("Email validation failed for", email)
      t.Log("Should Return", shouldMatch)
      t.Fail()
    }
  }
}

func TestHasEssentials(t *testing.T){
  t.Log("Testing helpers.HasEssentials")
  r := &request.Request{}
  r.Body = make(map[string][]string)
  if HasEssentials(r) != false {
    t.Fail()
  }
  // empty field
  r.Body["to"] = []string{}
  r.Body["from"] = []string{}
  t.Log("Empty to, from, mus fail")
  if HasEssentials(r) != false {
    t.Fail()
  }
  t.Log("Invalid email, must fail")
  // invalid email
  r.Body["to"] = []string{"t@.com"}
  r.Body["from"] = []string{"t@t.com"}
  if HasEssentials(r) != false {
    t.Fail()
  }
  r.Body["subject"] = []string{}
  r.Body["body"] = []string{}
  t.Log("Empty subject and body, must fail")
  if HasEssentials(r) != false {
    t.Fail()
  }
  r.Body["to"] = []string{"t@t.com"}
  r.Body["subject"] = []string{"Hello World"}
  r.Body["body"] = []string{"Mah message"}
  t.Log("Complete body, must pass")
  if HasEssentials(r) != true {
    t.Fail()
  }
}