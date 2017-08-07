/*
  This file contains the generic functions for handling different providers
  listener's and worker's task.
  ProcessMail & ProcessLog are identical functions and can be further merged
  into a single one but that would make managing the code more complex.
*/
package worker

import(
  "rabbit"
  "models"
  "helpers"
  "strings"
  "strconv"
  "fmt"
  "time"
  "doggo"
  "encoding/json"
  "github.com/streadway/amqp"
  "github.com/gocql/gocql"
)
const MAX_RETRY_COUNT = 6
type GetEmailVars func(map[string]string, map[string]string) (string, error)
type SendRequest func(*string) (int, bool, error)
/*
  Generic Function to dispatch email batches
  @params:
    packet : Rabbitmq packet
    provider: Provider name i.e. sendgrid, mailgun
    routingKey: Routing key intials i.e. "sglog.", "mglog.*"
    channel: The channel to which it has to retry publishing
    getEmailVarsFunc: A function which will return the body that has to be posted in request
    sendRequestFunc: A send helper function which dispatches the request to the known client
*/

func ProcessMail(packet amqp.Delivery, currentChannel string, getEmailVarsFunc GetEmailVars, sendRequestFunc SendRequest){
  // our batch_id is inside the routing key itself
  var retryCount int = 0
  var parts = strings.Split(packet.RoutingKey, ".")
  batch_id := parts[1]
  // This is a safe guard which helps us in preventing
  // infinite to-and-fro calls to different queues of providers
  if len(parts) == 4 && parts[2] == "retry" {
    retryCount, _ = strconv.Atoi(parts[3])
  }
  fmt.Println("Recieved message, processing", batch_id)
  id, err := gocql.ParseUUID(batch_id)
  if err != nil {
    // not a valid uuid
    fmt.Println("Not a valid UUID")
    packet.Ack(false)
    return
  }
  batch := models.Batch{Batch_id: id}
  success, err := batch.Exists()
  if success {
    if batch.Options["isBulk"] == "" {
        batch.Options["isBulk"] = "false"
    }
   var customArgs = map[string]string{"batch_id": batch_id}
   // ideally this should never, ever! fail
   body, err := getEmailVarsFunc(batch.Options, customArgs)
   // error Object that will be pushed in the DB in failure
   errorObject := map[string]interface{}{
    "status": "failed",
    "last_updated": time.Now(),
    "reason": "Bad Request",
    "description": "Please check the input body that you have posted",
    }
   if err != nil {
    _, err := batch.Update(errorObject)
    if err != nil {
      // DB failure 😔
      // put back in queue for someone else to do it
      packet.Nack(false, true)
      return
    }
    packet.Ack(false)
    return
   }
   
   status, requeue, err := sendRequestFunc(&body)
   if requeue == true {
    packet.Nack(false, true)
    return
   }

   // check if SG server accepted our request or not
   if status == 202 || status == 200 {
    doggo.AddDoggoMetric(currentChannel + ".200")
    _, err := batch.Update(map[string]interface{}{
      "status": "queued",
      "last_updated": time.Now(),
      })
    packet.Ack(false)
    if err != nil {
      // todo: Log error and push into a different queue to save it later
      fmt.Println(err)
    }
   } else {
    // Bad request, that means the format we sent the message in
    // was wrong, we should ideally notify the user as well as log
    // this thing
    // mailgun when rate limits also throws 400 :(
    if status == 400 {
      doggo.AddDoggoMetric(currentChannel + ".400")
      doggo.DoggoEvent(currentChannel + " failed", err , true)
      _, err := batch.Update(errorObject)
      if err != nil {
        // DB failure 😔
        // put in db_batch queue for someone else to do it
        packet.Ack(false)
        return
      }
      packet.Ack(false)
      return
    }
    doggo.AddDoggoMetric(currentChannel + ".502")
    // In other cases we can safely push them to another channel
    worker, remaining := helpers.GetNextChannel(currentChannel, string(packet.Body))
    routingKey := packet.RoutingKey
    if worker == nil && helpers.CanRetry(MAX_RETRY_COUNT, retryCount, &routingKey) == false {
      // we have exhausted all the providers, drop the packet
      doggo.AddDoggoMetric(currentChannel + ".permanenetfail")
      _, err := batch.Update(map[string]interface{}{
          "status": "failed",
        })
      if err != nil {
        // publish to db queue
      }
      packet.Ack(false)
      return
    }
    // we can still try as MAX_RETRY_COUNT hasn't been exhausted
    if worker == nil {
      worker, remaining = helpers.GetNextChannel(currentChannel, remaining)
    }
    routingKey = worker.Namespace + batch_id + ".retry."
    doggo.AddDoggoMetric(worker.Namespace + "retry")
    if helpers.CanRetry(MAX_RETRY_COUNT, retryCount, &routingKey) {
      // Publish it for requeue
      worker.Channel.Publish(routingKey, []byte(remaining))
    }
    packet.Ack(false)
   }
  } else {
    // batch doesn't exists, drop the packet, move on
    packet.Ack(false)
  }
}

/*
  Generic Function to process logs published for retrial
  @params:
    packet : Rabbitmq packet
    provider: Provider name i.e. sendgrid, mailgun
    routingKey: Routing key intials i.e. "sglog.", "mglog.*"
    channel: The channel to which it has to retry publishing
    getEmailVarsFunc: A function which will return the body that has to be posted in request
    sendRequestFunc: A send helper function which dispatches the request to the known client
*/
func ProcessLog(packet amqp.Delivery, provider string, getEmailVarsFunc GetEmailVars, sendRequestFunc SendRequest){
  Log, Batch, log_id, batch_id, retryCount, nack, ack, err := helpers.LogListener(packet)
  // packet data not valid
  if err != nil {
    // negative acknowledge
    if nack && ack {
      packet.Nack(false, true)
      return
    }
    // drop the packet
    if nack && !ack {
      packet.Nack(false, false)
      return
    }
    // acknowledge and drop
    if ack {
      packet.Ack(false)
    }
    return
  }
  // singular email entity
  var options = map[string]string{
    "to": Log.Email,
    "from": Batch.Options["from"],
    "name": Batch.Options["name"],
    "subject": Batch.Options["subject"],
    "content": Batch.Options["content"],
  }
  var customArgs = map[string]string{"batch_id": batch_id.String(), "log_id": Log.Log_id.String()}
  body, err := getEmailVarsFunc(options, customArgs)
  // This should never ever fails, but the state will become undefined here after
  // so mark this as failed
  if err != nil {
    _, err := Log.Update(map[string]interface{}{
      "last_update": time.Now(),
      "status": "failed",
      })
    if err != nil {
      // DB failure 😔
      // put back in queue for someone else to do it
      packet.Nack(false, true)
      return
    }
    packet.Ack(false)
    return
  }
  statusCode, requeue, err := sendRequestFunc(&body)
  // RequestClient failed, reque for someone else
  if requeue == true {
    packet.Nack(false, true)
    return
  }
  var status = Log.Status
  status[provider] = true
  // update object
  var updateMap = map[string]interface{}{
    "state": "dispatched",
    "status": status,
    "last_update": time.Now(),
  }
  // success! Save the new status
  if statusCode == 202 || statusCode == 200 {
    doggo.AddDoggoMetric(provider + ".200")
    packet.Ack(false)
    // mark sendgrid as used provider
    _, err := Log.Update(updateMap)
    if err != nil {
      // failed db update, put in db_log queue
      data, _ := json.Marshal(updateMap)
      rabbit.LogDB.Publish("logdb.log." + log_id.String(), data)
    }
    return
  } else {
    // Bad JSON format, our fault, this can't be fixed on our end
    // we would need human intervention in this case
    // mark it as failed and exit
    if statusCode == 400 {
      doggo.AddDoggoMetric(provider + ".400")
      doggo.DoggoEvent(provider + " failed", err , true)
      errorMap := map[string]interface{}{
        "state": "failed",
        "status": status,
        "last_update": time.Now(),
      }
      _, err := Log.Update(errorMap)
      if err != nil {
        // DB failure 😔
        // failed db update, put in db_log queue
        data, _ := json.Marshal(updateMap)
        rabbit.LogDB.Publish("logdb.log." + log_id.String(), data)
      }
      packet.Ack(false)
    } else {
      // Other errors are server errors of provider's server
      // we can requeue the message with retry flag
      // and retry for MAX_RETRY_COUNT
      // In other cases we can safely push them to another channel
      doggo.AddDoggoMetric(provider + ".502")
      worker, remaining := helpers.GetNextChannel(provider, string(packet.Body))
      routingKey := packet.RoutingKey
      if worker == nil && helpers.CanRetry(MAX_RETRY_COUNT, retryCount, &routingKey) == false {
        // we have exhausted all the providers, drop the packet
        doggo.AddDoggoMetric(provider + ".permanenetfail")
        _, err := Log.Update(map[string]interface{}{
            "state": "failed",
          })
        if err != nil {
          // publish to db queue
        }
        packet.Ack(false)
        return
      }
      // we can still try as MAX_RETRY_COUNT hasn't been exhausted
      if worker == nil {
        worker, remaining = helpers.GetNextChannel(provider, remaining)
      }
      fmt.Println(remaining, worker)
      routingKey = "log." + worker.Namespace + log_id.String() + ".retry."
      doggo.AddDoggoMetric(worker.Namespace + "retry")
      if helpers.CanRetry(MAX_RETRY_COUNT, retryCount, &routingKey) {
        // Publish it for requeue
        worker.Channel.Publish(routingKey, []byte(remaining))
      }
      packet.Ack(false)
    }
  }
}