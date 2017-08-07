package helpers

import (
	"testing"
  "rabbit"
)

func TestCanRetry(t *testing.T){
  var retryRoutingKey string = "mg"
  if CanRetry(0, 0, &retryRoutingKey) != false {
    t.Fail()
  }
  if CanRetry(1, 0, &retryRoutingKey) != true && retryRoutingKey != "mg.1" {
    t.Log("Routing Key must be mg.1")
    t.Fail()
  }
  if CanRetry(5, 4, &retryRoutingKey) != true && retryRoutingKey != "mg.5" {
    t.Log("Routing Key must be mg.5", retryRoutingKey)
    t.Fail()
  }
}
/*
  Need rabbitmq to be turned on in order to pass this test
*/
func TestNextChannelToTry(t *testing.T){
  rabbit.EmailWorkers = make(map[string]*rabbit.WorkerList)
  rabbit.EmailWorkers["sendgrid"] = &rabbit.WorkerList{Namespace: "sg."}
  rabbit.EmailWorkers["mailgun"] = &rabbit.WorkerList{Namespace: "mg."}

  if NextChannelToTry(map[string]bool{}) == nil {
    t.Log("Must return a worker for empty tried list")
    t.Fail()
  }
  worker := NextChannelToTry(map[string]bool{"sendgrid": true})
  if  worker == nil || worker.Namespace != "mg." {
    t.Log("Must return mail gun worker", worker)
    t.Fail()
  }
}

func TestGetNextChannel(t *testing.T){
  worker, _ := GetNextChannel("mailgun", "")
  if worker == nil || worker.Namespace != "sg." {
    t.Log("Should return sendgrid worker")
    t.Fail()
  }
  worker, remaining := GetNextChannel("sendgrid", "mailgun,sendgrid")
  if worker == nil || worker.Namespace != "mg." || remaining != "sendgrid,sendgrid" {
    t.Log("Mailgun worker should have been returned and remaining should be sendgrid")
    t.Fail()
  }
  worker, remaining = GetNextChannel("sendgrid", "sendgrid")
  if worker == nil || worker.Namespace != "sg." || remaining != "sendgrid" {
    t.Log("Mailgun worker should have been returned and remaining should be sendgrid")
    t.Fail()
  }
  worker, remaining = GetNextChannel("sendgrid", "")
  if worker == nil || worker.Namespace != "mg." || remaining != "sendgrid" {
    t.Log("Mailgun worker should have been returned and remaining should be sendgrid")
    t.Fail()
  }
}