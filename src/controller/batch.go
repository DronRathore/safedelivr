package controller

import (
  "time"
  "models"
  "helpers"
  "rabbit"
  "doggo"
  "github.com/gocql/gocql"
  request "github.com/DronRathore/goexpress/request"
  response "github.com/DronRathore/goexpress/response"
  express "github.com/DronRathore/goexpress"
)

// Fetches data of a batch, provided the api_key
func BatchIndex(req *request.Request, res *response.Response, next func()){
  var batch models.Batch
  var user models.User
  var user_id gocql.UUID
  var batch_id gocql.UUID
  var success bool
  var userData *models.UserRedis
  var exists bool
  var err error

  if len(req.Params["id"]) < 36 {
      goto SendIdError
  }
  if IsLoggedIn(res) == false {
    if len(req.Query["user_id"]) == 0 || len(req.Params["id"]) == 0 {
      goto Send403
    }
    batch_id, err = gocql.ParseUUID(req.Params["id"])
    if err != nil {
      doggo.DoggoEvent("ParseUUID failed", err, true)
      goto Send500
    }
    user_id, err = gocql.ParseUUID(req.Query["user_id"][0])
    if err != nil {
      doggo.DoggoEvent("ParseUUID failed", err, true)
      goto Send500
    }
    if len(req.Query["api_key"]) == 0 {
      doggo.AddDoggoMetric("failed.api_key")
      goto Send403
    }
    user = models.User{User_Id: user_id}
    success, err = user.Exists()
    // user not found
    if err != nil || !success {
      goto Send403
    }
    if user.Api_Key != req.Query["api_key"][0] {
      goto Send403
    }
  } else {
    userData = res.Locals["user"].(*models.UserRedis)
    user_id, err = gocql.ParseUUID(userData.UUID)
    batch_id, err = gocql.ParseUUID(req.Params["id"])
    if err != nil {
      goto Send500
    }
  }
  batch = models.Batch{Batch_id: batch_id}

  exists, err = batch.Exists()
  if err != nil {
    goto Send500
  }
  if exists == true {
    if batch.User_id != user_id {
      goto Send403
    }
    // return the map
    doggo.AddDoggoMetric("server.200")
    res.JSON(batch.GetMap())
    return
  } else {
    doggo.AddDoggoMetric("batch.404")
    res.Header.SetStatus(404)
    res.JSON(map[string]string{"error": "Batch doesn't exists"})
    return
  }
  Send403:
    doggo.AddDoggoMetric("auth.403")
    res.Header.SetStatus(403)
    res.JSON(map[string]string{"error": "Not a valid api_key or user_id"})
  SendIdError:
    doggo.AddDoggoMetric("body.400")
    res.Header.SetStatus(400)
    res.JSON(map[string]string{"error": "Invalid Batch ID"})
    return
  Send500:
    doggo.AddDoggoMetric("server.500")
    res.Header.SetStatus(500)
    res.JSON(map[string]string{"error": "Internal Server Error"})
    return
}
/*
  
*/
func BatchList(req *request.Request, res *response.Response, next func()){
  var user_id gocql.UUID
  var userData *models.UserRedis
  var err error

  if IsLoggedIn(res) == false {
    goto Send403
  } else {
    userData = res.Locals["user"].(*models.UserRedis)
    user_id, err = gocql.ParseUUID(userData.UUID)
    docs := models.FetchBatches(user_id, 20)
    doggo.AddDoggoMetric("server.200")
    res.JSON(docs)
    return
  }
  Send403:
    doggo.AddDoggoMetric("auth.403")
    res.Header.SetStatus(403)
    res.JSON(map[string]string{"error": "Not a valid api_key or user_id"})
  Send500:
    doggo.AddDoggoMetric("server.500")
    res.Header.SetStatus(500)
    res.JSON(map[string]string{"error": "Internal Server Error"})
    return
}
/*
  Retrieves all the logs associated to a batch, a batch can have multiple logs
  attached to it, each corresponding to a individual recipient
*/
func BatchLogs(req *request.Request, res *response.Response, next func()) {
  var batch models.Batch
  var user models.User
  var batch_id gocql.UUID
  var user_id gocql.UUID
  var userData *models.UserRedis
  var exists bool
  var err error

  if len(req.Params["id"]) < 36 {
      goto SendIdError
  }
  if IsLoggedIn(res) == false {
    if len(req.Query["user_id"]) == 0 || len(req.Params["id"]) == 0 {
      goto Send403
    }
    batch_id, err = gocql.ParseUUID(req.Params["id"])
    if err != nil {
      goto Send500
    }
    user_id, err = gocql.ParseUUID(req.Query["user_id"][0])
    if err != nil {
      goto Send500
    }
    if len(req.Query["api_key"]) == 0 {
      goto Send403
    }
    user = models.User{User_Id: user_id}
    exists, err = user.Exists()
    // user not found
    if err != nil || !exists {
      goto Send403
    }
    if user.Api_Key != req.Query["api_key"][0] {
      goto Send403
    }
  } else {
    userData = res.Locals["user"].(*models.UserRedis)
    user_id, err = gocql.ParseUUID(userData.UUID)
    batch_id, err = gocql.ParseUUID(req.Params["id"])
    if err != nil {
      goto Send500
    }
  }
  batch = models.Batch{Batch_id: batch_id}
  exists, err = batch.Exists()
  if err != nil {
    goto Send500
  }
  if exists == true {
    if batch.User_id != user_id {
      goto Send403
    }
    // batch exists, lets fetch the logs as per pagination
    var lastPage gocql.UUID
    if len(req.Query["page"]) != 0 {
      lastPage, _ = gocql.ParseUUID(req.Query["page"][0])
    }
    var logs = models.FetchLogs(batch_id, lastPage, "last_update", 20)
    doggo.AddDoggoMetric("server.200")
    res.JSON(logs)
    return
  } else {
    doggo.AddDoggoMetric("server.404")
    res.Header.SetStatus(404)
    res.JSON(map[string]string{"error": "Batch doesn't exists"})
    return
  }
  Send403:
    doggo.AddDoggoMetric("auth.403")
    res.Header.SetStatus(403)
    res.JSON(map[string]string{"error": "Not a valid api_key or user_id"})
  SendIdError:
    doggo.AddDoggoMetric("server.400")
    res.Header.SetStatus(400)
    res.JSON(map[string]string{"error": "Invalid Batch ID"})
    return
  Send500:
    doggo.AddDoggoMetric("server.500")
    res.Header.SetStatus(500)
    res.JSON(map[string]string{"error": "Internal Server Error"})
    return
}
// Create a new mail Batch for dispatch
func BatchCreate(req *request.Request, res *response.Response, next func()) {
  var user *models.User
  var batch *models.Batch
  var exists bool
  var success bool
  var err error
  var user_id gocql.UUID
  var concatedList string
  var length int
  var options map[string]string
  var userData *models.UserRedis
  
  // need these to go forward
  // for k, v := range req.Body {
    // fmt.Println(k, v)
  // }
  
  if IsLoggedIn(res) == false && (req.Body["api_key"] == nil || req.Body["user_id"] == nil || len(req.Body["user_id"]) == 0 || len(req.Body["api_key"]) == 0) {
    goto Send403
  }
  if IsLoggedIn(res) == true {
    userData = res.Locals["user"].(*models.UserRedis)
    user_id, err = gocql.ParseUUID(userData.UUID)
    user = &models.User{User_Id: user_id}
    req.Body["user_id"] = []string{user_id.String()}
  } else {
    user = &models.User{Api_Key: req.Body["api_key"][0]}
  }
  exists, err = user.Exists()
  if err != nil {
    goto Send500
  }
  if exists == false {
    // no such user exits
    goto Send403
  }
  if req.Body["api_key"] == nil && IsLoggedIn(res) == true {
    req.Body["api_key"] = []string{user.Api_Key}
  }
  // check if the user_id is valid for this api_key
  if IsLoggedIn(res) == false && user.GetId() != req.Body["user_id"][0] {
    goto Send403
  }
  // valid request, lets process
  if helpers.HasEssentials(req) == false {
    goto Send400
  }
  options = make(map[string]string)
  options["from"] = req.Body["from"][0]
  options["subject"] = req.Body["subject"][0]
  options["content"] = req.Body["body"][0]
  if req.Body["is_bulk"] != nil && req.Body["is_bulk"][0] == "true" {
    options["isBulk"] = "true"
  } else {
    options["isBulk"] = "false"
  }
  if req.Body["name"] != nil && len(req.Body["name"][0]) > 1 {
    options["name"] = req.Body["name"][0]
  } else {
    options["name"] = ""
  }
  concatedList = ""
  length = len(req.Body["to"])
  for _, email := range req.Body["to"] {
    if helpers.IsEmail(email) {
      if length != 1 {
        concatedList = concatedList + email + ","
      } else {
        concatedList = concatedList + email
      }
    }
    length = length - 1
  }
  // in case we missed it all
  if len(concatedList) == 0 {
    goto Send400
  }
  options["to"] = concatedList
  // push into the database and then publish it on queue
  batch = models.NewBatch()
  user_id, _ = gocql.ParseUUID(req.Body["user_id"][0])
  success, err = batch.Create(map[string]interface{}{
    "subject": req.Body["subject"][0],
    "user_id": user_id,
    "status" : "Acknowledged",
    "options": options,
    "created_at": time.Now()})
  if err != nil || success == false {
    goto Send500
  }
  // update stats
  helpers.UpdateStats(batch.User_id, "queued")
  // publish to queue
  rabbit.BatchChannel.Publish("batch." + batch.GetId(), []byte(""))
  // todo: update quota counters
  // return an accepted status
  doggo.AddDoggoMetric("server.200")
  res.Header.SetStatus(202)
  res.JSON(map[string]interface{}{"status": "success", "batch_id": batch.GetId()})
  return

  // errors
  Send403:
    doggo.AddDoggoMetric("auth.403")
    res.Header.SetStatus(403)
    res.JSON(map[string]string{"error": "Access denied"})
    return
  Send400:
    doggo.AddDoggoMetric("server.400")
    res.Header.SetStatus(400)
    res.JSON(map[string]string{"error": "Wrong parameters sent"})
    return
  Send500:
    doggo.AddDoggoMetric("server.500")
    res.Header.SetStatus(500)
    res.JSON(map[string]string{"error": "Internal Server Error"})
}

var BatchController = func() interface{} {
  var Router = express.Router()
  Router.Post("/api/batch", BatchCreate)
  Router.Get("/api/batches", BatchList)
  Router.Get("/api/batch/:id", BatchIndex)
  Router.Get("/api/batch/:id/logs", BatchLogs)
  return *Router
}()
