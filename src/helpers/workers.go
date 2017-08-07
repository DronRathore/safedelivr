package helpers

import (
  "models"
  "rabbit"
  "strconv"
  "fmt"
  "errors"
  "strings"
  "github.com/streadway/amqp"
  "github.com/gocql/gocql"
)
func CanRetry(maxAllowed int, retryCount int, routingKey *string) bool {
  if retryCount > maxAllowed {
    return false
  }
  retryCountStr := strconv.Itoa(retryCount + 1)
  *routingKey = *routingKey + retryCountStr
  return true
}
/*
  Returns a provider which hasn't been used yet
*/
func NextChannelToTry(tried map[string]bool)(*rabbit.WorkerList){
  for channel, worker := range rabbit.EmailWorkers {
    if tried[channel] == false {
      return worker
    }
  }
  return nil
}
/*
  Helper for Amq Consumer to get next queue name to be used for retrial
*/
func GetNextChannel(currentChannel string, list string) (*rabbit.WorkerList, string) {
  fmt.Println("Getting next channel=>", list)
  if list == "" {
    list = ""
    for channelName, _ := range rabbit.EmailWorkers {
      // we don't want to prioritize same channel
      if channelName != currentChannel {
        list = list + channelName + ","
      }
    }
  }
  var parts = strings.Split(list, ",")
  if len(parts) == 0 {
    return nil, ""
  }
  // add the currentChannel at the end of the priority
  list = list[:len(list) - 1] + "," + currentChannel
  name := strings.Split(list, ",")[0]
  return rabbit.EmailWorkers[name], list[strings.Index(list, ",") + 1:]
}
/*
  A common helper to perform basic checks on a published key on Log queues
  This helper won't take any decision on the packet, avoiding any arbitary flaws
  @return
    Log: Log Model
    Batch: Batch Model
    log_id: UUID of Log Model
    batch_id: UUID of Batch Model
    retryCount: int
    Nack: Negative acknowledge a packet
    ACK: Positive acknowledge a packet
    error: error
*/
func LogListener(packet amqp.Delivery) (*models.Log, *models.Batch, gocql.UUID, gocql.UUID, int, bool, bool, error) {
  var emptyUUID gocql.UUID
  var retryCount int = 0

  var parts = strings.Split(packet.RoutingKey, ".")
  if len(parts) < 3 {
    // fake packet
    return nil, nil, emptyUUID, emptyUUID, 0, false, true, errors.New("Invalid UUID")
  }
  // is this a retry packet?
  if len(parts) == 5 && parts[3] == "retry" {
    retryCount, _ = strconv.Atoi(parts[3])
  }

  var id = parts[2]

  log_id, err := gocql.ParseUUID(id)
  if err != nil {
    // not a valid uuid
    fmt.Println("Not a valid UUID")
    return nil, nil, emptyUUID, emptyUUID, 0, false, true, errors.New("Invalid UUID")
  }

  var Log = models.Log{Log_id: log_id}
  success, err := Log.Exists()
  if err != nil {
    // cannot connect to DB?
    // put back in queue for some one else
    fmt.Println("Error: Log doesn't exists", err)
    return nil, nil, emptyUUID, emptyUUID, 0, true, true, errors.New("Db connection error")
  }
  // log doesn't exits 🤔
  if success == false {
    fmt.Println("Log doesn't exists")
    return nil, nil, emptyUUID, emptyUUID, 0, false, true, errors.New("Log doesn't exists")
  }
  // get the original content & subject to send
  var Batch = models.Batch{Batch_id: Log.Batch_id}
  success, err = Batch.Exists()
  if err != nil {
    // cannot connect to DB?
    // put back in queue for some one else
    fmt.Println("Error: Batch doesn't exists", err)
    return nil, nil, emptyUUID, emptyUUID, 0, true, true, errors.New("Db connection error")
  }
  // Batch doesn't exits 🤔
  if success == false {
    fmt.Println("Batch doesn't exists", Log.Batch_id)
    return nil, nil, emptyUUID, emptyUUID, 0, false, true, errors.New("Batch doesn't exists")
  }
  return &Log, &Batch, log_id, Batch.Batch_id, retryCount, false, false, nil
}