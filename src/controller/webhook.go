package controller

import (
  "encoding/json"
  "fmt"
  "time"
  "config"
  "strings"
  "strconv"
  "models"
  "helpers"
  "doggo"
  "io/ioutil"
  "github.com/gocql/gocql"
  request "github.com/DronRathore/goexpress/request"
  response "github.com/DronRathore/goexpress/response"
  express "github.com/DronRathore/goexpress"
)

///////////////// Webhooks ////////////////////////

/* 
  Updates Sendgrid status
  Sendgrid is by far the best way to handle emails
  The status varies with granular details which helps in taking
  decisions properly. Sendgrid posts an array of JSON events.
*/
func BatchUpdateSGStatus(req *request.Request, res *response.Response, next func()) {
  username, password, ok := req.GetRaw().BasicAuth()
  if !ok {
    res.Header.SetStatus(403)
    res.JSON(map[string]interface{}{
      "error": "Access denied",
    })
    return
  }
  // Validate Auth headers
  if username != config.Configuration.Sendgrid.Username || password != config.Configuration.Sendgrid.Password {
    res.Header.SetStatus(403)
    res.JSON(map[string]interface{}{
      "error": "Access denied",
    })
    return
  }
  // valid request, lets process the event
  // read and parse the JSON body
  data, err := ioutil.ReadAll(req.GetRaw().Body)
  if err != nil {
    res.Header.SetStatus(400)
    doggo.DoggoEvent("Cannot read sendgrid JSON", err, true)
    return
  }
  var sendgridJson []map[string]interface{}
  err = json.Unmarshal(data, &sendgridJson)
  if err != nil {
    res.Header.SetStatus(400)
    doggo.DoggoEvent("Parse sendgrid JSON failed", err, true)
    return
  }
  // empty events array pushed
  if len(sendgridJson) == 0 {
    res.Header.SetStatus(200)
    res.End()
    return
  }
  // Return the response to SG, and then process the events
  // so as to avoid timeouts
  res.Write("Success").End()

  // loop over all the events that are pushed
  for _, event := range sendgridJson {
    var batchId gocql.UUID
    var logId gocql.UUID
    var emptyId gocql.UUID
    var timestamp time.Time = time.Now()

    eventName := strings.ToLower(event["event"].(string))
    if event["batch_id"] == nil {
      // break, we expect a batch id
    } else {
      batchId, _ = gocql.ParseUUID(event["batch_id"].(string))
    }

    if event["log_id"] != nil {
      logId, _ = gocql.ParseUUID(event["log_id"].(string))
    }

    var log *models.Log = nil
    var shouldRetry bool = false
    var updateObject map[string]interface{}
    switch(eventName){
      case "processed" :
        updateObject = map[string]interface{}{"state": "sent"}
        break;
      case "delivered" : 
        updateObject = map[string]interface{}{"state": "delivered"}
      break;
      case "deferred"  :
        updateObject = map[string]interface{}{"state": "deferred"}
        shouldRetry = true
        break;
      case "bounce"    : 
        updateObject = map[string]interface{}{"state": "bounced"}
        shouldRetry = true
      break;
      case "dropped"   : 
        updateObject = map[string]interface{}{"state": "dropped"}
        shouldRetry = false
      break;
      default: continue
    }

    if event["timestamp"] != nil {
      timestamp = time.Unix(int64(event["timestamp"].(float64)), 0)
    }
    // set the updated timestamp
    updateObject["last_update"] = timestamp

    batch := &models.Batch{Batch_id: batchId}
    success, err := batch.Exists()
    if err != nil {
      // cannot process the event
      // todo: Push to a DB rabbit queue
      doggo.DoggoEvent("Batch Search failed", "BatchId:" + batchId.String() + "\n" + err.Error() , true)
      continue
    }
    
    if (success == false){
      // batch doesn't exists
      // Ack the request and move on
      continue
    }
    // fetch the email log doc
    if logId != emptyId {
      log = &models.Log{Batch_id: batchId, Log_id: logId}
      success, _ := log.Exists()
      if success == true {
        // this email is done
        // mark sendgrid as been used
        var status = log.GetMap()
        status["sendgrid"] = true
        updateObject["status"] = status
        if shouldRetry == false {
          // Lets update the status and exit
          log.Update(updateObject)
          continue
       }
      } else {
        // drop the packet
        // log_id doesn't exists, dangling packet
        continue
     }
    }
    // this is a retrial/first info event request
    if log == nil {
      // create a log
      fmt.Println("Creating a log")
      log = &models.Log{}
      _, err := log.Create(map[string]interface{}{
        "state"  : "queued",
        "status" : map[string]bool{"sendgrid": true},
        "batch_id": batchId,
        "user_id": batch.User_id,
        "email": event["email"].(string),
        "created_at": time.Now(),
        "last_update": timestamp,
      })
      fmt.Println("Error=>", err)
      if err != nil {
        doggo.DoggoEvent("Log Creation failed", "BatchId:" + batchId.String() + "\n" + err.Error() , true)
      }
    }
    fmt.Println("Log Created")
    // successfully delivered the packet, save the log
    // and continue
    if shouldRetry == false {
      doggo.AddDoggoMetric("sendgrid.success")
      helpers.UpdateStats(batch.User_id, "success")
      _, err := log.Update(updateObject)
      fmt.Println("Log Updated", err)
      continue
    }
    fmt.Println("Trying the next worker")
    // else our mail has been dropped lets see what we can do
    // pick the one which we haven't tried
    doggo.AddDoggoMetric("sendgrid.failed")
    helpers.UpdateStats(batch.User_id, "failed")
    worker := helpers.NextChannelToTry(log.Status)
    if worker != nil {
      // use this worker to retry sending the email
      routingKey := "log." + worker.Namespace + log.Log_id.String()
      fmt.Println("RoutingKey", routingKey)
      // push it into the queue
      worker.Channel.Publish(routingKey, nil)
    } else {
      // exhausted all the trials for this email
      // drop it permanently
      fmt.Println("Le'me just update and exit")
      log.Update(updateObject)
    }
  }
}

/*
  MailGun Webhook
  Mailgun service only provides delivered, bounced and hardfail status
  so we can only take actions in case of a bounced email only.
*/
func BatchUpdateMGStatus(req *request.Request, res *response.Response, next func()) {
  var batchId gocql.UUID
  var logId gocql.UUID
  var emptyId gocql.UUID
  var email string
  var updateObject map[string]interface{}
  var err error
  var batch *models.Batch
  var log *models.Log = nil
  var success bool
  var boundary string
  var timestamp time.Time
  // Mailgun has this strange way of sending status updates
  // if it is a failed event than you get a form-data else a simple
  // post form is sent in case of delivered status
  if helpers.IsMultipart(req.Header["content-type"], &boundary) {
    form := helpers.ReadMultiPartForm(boundary, req.GetRaw().Body)
    if form == nil {
      res.Header.SetStatus(400)
      res.Write("Bad Request").End()
      return
    }
    // inject the form-data values
    req.Body = form.Value
  }
  // validate the request, should have valid auth header
  if helpers.IsValidMGRequest(req) == false {
    res.Header.SetStatus(403)
    res.Write("Access Denied").End()
    return
  }
  // simple checks
  if len(req.Body["event"]) == 0 || len(req.Body["recipient"]) == 0 || len(req.Body["batch-id"]) == 0 {
    res.Header.SetStatus(400)
    res.Write("Bad Request").End()
    return
  }
  // Cool, it is a valid webhook request from MG
  // process it.
  batch_id_data := req.Body["batch-id"][0]
  if batch_id_data == "" {
    // batch_id not present, drop the request
    res.Header.SetStatus(400)
    res.Write("Bad Request").End()
    return
  }
  // parse the batch_id
  batchId, err = gocql.ParseUUID(batch_id_data)
  if err != nil {
    // we messed up! Why would you send a bad JSON dron?
    res.Header.SetStatus(400)
    res.Write("Bad Request").End()
    return
  }
  // see if we have a log_id
  if len(req.Body["log-id"]) == 1 {
    logId, err = gocql.ParseUUID(req.Body["log-id"][0])
  }
  email = req.Body["recipient"][0]
  var eventName = req.Body["event"][0]
  var shouldRetry bool = false
  // send success to the client and then process it
  res.Write("success").End()

  switch(eventName){
    case "delivered" : 
      updateObject = map[string]interface{}{"state": "delivered"}
    break;
    case "bounced"    : 
      updateObject = map[string]interface{}{"state": "bounced"}
      shouldRetry = true
    break;
    case "hardfail"   : 
      updateObject = map[string]interface{}{"state": "dropped"}
      shouldRetry = false
    break;
    default: return // we can add further click/open events but OOS
  }
  // retrieve the timestamp
  if len(req.Body["timestamp"]) != 0 {
    val, _ := strconv.Atoi(req.Body["timestamp"][0])
    timestamp = time.Unix(int64(val), 0)
  }
  // set the updated timestamp
  updateObject["last_update"] = timestamp

  batch = &models.Batch{Batch_id: batchId}
  success, err = batch.Exists()
  if err != nil {
    // cannot process the event
    // todo: Push to a DB rabbit queue
    doggo.DoggoEvent("Batch Search failed", "BatchId:" + batchId.String() + "\n" + err.Error() , true)
    return
  }
  
  if (success == false){
    // batch doesn't exists
    // Ack the request and move on
    return
  }
  // fetch the email log doc
  if logId != emptyId {
    log = &models.Log{Batch_id: batchId, Log_id: logId}
    success, _ = log.Exists()
    if success == true {
      // this email is done
      // mark mailgun as been used
      var status = log.GetMap()
      status["mailgun"] = true
      updateObject["status"] = status
      if shouldRetry == false {
        // Lets update the status and exit
        log.Update(updateObject)
        return
     }
    } else {
      // drop the packet
      // log_id doesn't exists, dangling packet
      return
   }
  }
  // this is a retrial mail
  if log == nil {
    // create a log
    log = &models.Log{}
    _, err = log.Create(map[string]interface{}{
      "state"  : "queued",
      "status" : map[string]bool{"mailgun": true},
      "batch_id": batchId,
      "user_id": batch.User_id,
      "email": email,
      "created_at": time.Now(),
      "last_update": timestamp,
    })
    if err != nil {
      doggo.DoggoEvent("Log Creation failed", "BatchId:" + batchId.String() + "\n" + err.Error() , true)
    }
  }
  // successfully delivered the packet, save the log
  // and return
  if shouldRetry == false {
    helpers.UpdateStats(batch.User_id, "success")
    doggo.AddDoggoMetric("mailgun.success")
    _, err = log.Update(updateObject)
    return
  }
  fmt.Println("Trying the next worker")
  doggo.AddDoggoMetric("mailgun.failed")
  helpers.UpdateStats(batch.User_id, "failed")
  // else our mail has been dropped lets see what we can do
  // pick the one which we haven't tried
  worker := helpers.NextChannelToTry(log.Status)
  if worker != nil {
    // use this worker to retry sending the email
    routingKey := "log." + worker.Namespace + log.Log_id.String()
    fmt.Println("RoutingKey", routingKey)
    // push it into the queue
    worker.Channel.Publish(routingKey, nil)
  } else {
    // exhausted all the trials for this email
    // drop it permanently
    fmt.Println("Le'me just update and exit")
    log.Update(updateObject)
  }
}

var WebhookController = func() interface{} {
  var Router = express.Router()
  Router.Post("/api/webhook/sendgrid", BatchUpdateSGStatus)
  Router.Post("/api/webhook/mailgun", BatchUpdateMGStatus)
  return *Router
}()
