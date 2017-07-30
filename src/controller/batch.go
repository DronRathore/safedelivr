package controller

import (
  "models"
  "encoding/json"
  "fmt"
  "time"
  "config"
  "crypto/sha256"
  "crypto/hmac"
  "encoding/hex"
  "github.com/gocql/gocql"
  request "github.com/DronRathore/goexpress/request"
  response "github.com/DronRathore/goexpress/response"
  express "github.com/DronRathore/goexpress"
)

// check whether the given batch request has valid headers
func validBatch(batchObject map[string]interface{}) bool {
  if batchObject["to"] == nil || batchObject["from"] == nil || batchObject["subject"] == nil || batchObject["body"] == nil {
    return false
  }
  return true
}
// Fetches data of a batch, provided the api_key
func BatchIndex(req *request.Request, res *response.Response, next func()){
  var batch models.Batch
  var userData *models.UserRedis
  var exists bool
  var err error

  if len(req.Params["id"]) < 36 {
      goto SendIdError
  }
  if IsLoggedIn(res) == false {
    batch_id, err := gocql.ParseUUID(req.Params["id"])
    if err != nil {
      goto Send500
    }
    user_id, err := gocql.ParseUUID(req.Query["user_id"][0])
    if err != nil {
      goto Send500
    }
    batch = models.Batch{Batch_id: batch_id, User_id: user_id, Api_key: req.Query["api_key"][0]}
  } else {
    userData = res.Locals["user"].(*models.UserRedis)
    user_id, err := gocql.ParseUUID(userData.UUID)
    batch_id, err := gocql.ParseUUID(req.Params["id"])
    if err != nil {
      goto Send500
    }
    batch = models.Batch{Batch_id: batch_id, User_id: user_id}
  }
  exists, err = batch.Exists()
  if err != nil {
    goto Send500
  }
  if exists == true {
    // return the map
    res.JSON(batch.GetMap())
    return
  } else {
    res.Header.SetStatus(404)
    res.JSON(map[string]string{"error": "Batch doesn't exists"})
    return
  }
  SendIdError:
    res.Header.SetStatus(400)
    res.JSON(map[string]string{"error": "Invalid Batch ID"})
    return
  Send500:
    res.Header.SetStatus(500)
    res.JSON(map[string]string{"error": "Internal Server Error"})
    return
}
func hasEssentials(req *request.Request) bool {
  if req.Body["from"] == nil || len(req.Body["from"]) == 0 {
    return false
  }
  if req.Body["to"] == nil || len(req.Body["to"]) == 0 {
    return false
  }
  if req.Body["subject"] == nil || len(req.Body["subject"]) == 0 {
    return false
  }
  if req.Body["body"] == nil || len(req.Body["body"]) == 0 {
    return false
  }
  return true
}

// Create a new mail Batch for dispatch
func BatchCreate(req *request.Request, res *response.Response, next func()) {
  var user *models.User
  var batch *models.Batch
  var exists bool
  var success bool
  var err error
  var user_id gocql.UUID
  // var customOptions map[string]interface{}
  
  // need these to go forward
  // for k, v := range req.Body {
    // fmt.Println(k, v)
  // }
  if req.Body["api_key"] == nil || req.Body["user_id"] == nil || len(req.Body["user_id"]) == 0 || len(req.Body["api_key"]) == 0{
    goto Send403
  }

  user = &models.User{Api_Key: req.Body["api_key"][0]}
  exists, err = user.Exists()
  if err != nil {
    goto Send500
  }

  if exists == false {
    // no such user exits
    goto Send403
  }
  // check if the user_id is valid for this api_key
  if user.GetId() != req.Body["user_id"][0] {
    goto Send403
  }
  // valid request, lets process
  if hasEssentials(req) == false {
    goto Send400
  }
  
  // if req.Body["batch"] != nil && req.Body["batch"][0] == "true" {
    // customOptions = getCustomIds(req.Body["to"][0])
  // }

  // push into the database and then publish it on queue
  batch = models.NewBatch()
  user_id, _ = gocql.ParseUUID(req.Body["user_id"][0])
  success, err = batch.Create(map[string]interface{}{
    "subject": req.Body["subject"][0],
    "body": req.Body["body"][0],
    "user_id": user_id,
    "api_key": req.Body["api_key"][0],
    "status" : "Acknowledged",
    // "options": customOptions,
    "created_at": time.Now()})
  if err != nil || success == false {
    fmt.Println(err)
    goto Send500
  }
  // publish to queue
  // todo: update quota counters
  // return an accepted status
  res.Header.SetStatus(202)
  res.JSON(map[string]interface{}{"status": "success", "batch_id": batch.GetId()})
  return

  // errors
  Send403:
    res.Header.SetStatus(403)
    res.JSON(map[string]string{"error": "Access denied"})
    return
  Send400:
    res.Header.SetStatus(400)
    res.JSON(map[string]string{"error": "Wrong parameters sent"})
    return
  Send500:
    res.Header.SetStatus(500)
    res.JSON(map[string]string{"error": "Internal Server Error"})
}

///////////////// Webhooks ////////////////////////

// Updates Sendgrid status
func BatchUpdateSGStatus(req *request.Request, res *response.Response, next func()) {

}

func isValidMGRequest(req *request.Request) bool {
  if req.Body["timestamp"] == nil || req.Body["token"] == nil || req.Body["signature"] == nil {
    return false
  }
  var candidate = req.Body["timestamp"][0] + req.Body["token"][0]
  var hashKey = []byte(config.Configuration.MailGun.Api_Key)
  var hash = hmac.New(sha256.New, hashKey)
  hash.Write([]byte(candidate))
  return hex.EncodeToString(hash.Sum(nil)) == req.Body["signature"][0]
}

// Updates Mailgun statuses
func BatchUpdateMGStatus(req *request.Request, res *response.Response, next func()) {
  // validate the request
  var batch_id_data string
  var batchData map[string]interface{}
  var updateObject map[string]interface{}
  var err error
  var batch *models.Batch
  var exists bool
  var success bool
  if isValidMGRequest(req) == false {
    goto Send403
  }
  // is a valid webhook request from MG
  
  batch_id_data = req.Body["custom variables"][0]
  if batch_id_data == "" {
    // batch_id not present, drop the request
    goto Send400
  }
  // batch_id will be in json string, parse it
  batchData = make(map[string]interface{})
  err = json.Unmarshal([]byte(batch_id_data), &batchData)
  if err != nil {
    // we messed up! Why would you send a bad JSON dron?
    goto Send400
  }
  // update the batch status
  if batchData["batch_id"] == nil {
    // invalid batch_id
    goto Send400
  }
  batch = &models.Batch{Batch_id: batchData["batch_id"].(gocql.UUID)}
  exists, err = batch.Exists()
  if err != nil {
    goto Send500
  }

  if exists == false {
    // Mail with ghost batch id found
    // ghosts not allowed
    goto Send403
  }
  updateObject = make(map[string]interface{})
  updateObject["status"] = req.Body["event"][0]
  updateObject["description"] = req.Body["description"][0]
  updateObject["code"] = req.Body["code"][0]
  success, err = batch.Update(updateObject)
  if err != nil || success == false {
    goto Send500
  }
  // todo: Trigger user registered webhooks
  // push to Amq 
  res.JSON(map[string]string{"status": "success"})
  return

  // errors
  Send403:
    res.Header.SetStatus(403)
    res.JSON(map[string]string{"error": "Access denied"})
    return
  Send400:
    res.Header.SetStatus(400)
    res.JSON(map[string]string{"error": "Wrong parameters sent"})
    return
  Send500:
    res.Header.SetStatus(500)
    res.JSON(map[string]string{"error": "Internal Server Error"})
}


var BatchController = func() interface{} {
  var Router = express.Router()
  Router.Post("/api/batch", BatchCreate)
  Router.Get("/api/batch/:id", BatchIndex)
  Router.Post("/api/batch/webhook/sendgrid", BatchUpdateSGStatus)
  Router.Post("/api/batch/webhook/mailgun", BatchUpdateMGStatus)
  return *Router
}()
