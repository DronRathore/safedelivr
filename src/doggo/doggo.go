/*
  package doggo provides datadog's extension to send
  good metrics
*/
package doggo

import(
  datadog "github.com/DataDog/datadog-go/statsd"
  "fmt"
  "context"
  "time"
  "sync"
  "config"
)
var DoggoClient *datadog.Client
var counters map[string]int64
var lock *sync.Mutex

func DialDoggo(){
  var err error
  DoggoClient, err = datadog.New(config.Configuration.Datadog.Connstr)
  if err != nil {
    fmt.Println("Doggo cannot dial! Poor doggo died!", err)
    panic(err)
  }
  // namespace for go app
  DoggoClient.Namespace = "safego."
  lock = &sync.Mutex{}
  go DoggoSeive()
}
// A goodboy's event with all fellow doggos!
func DoggoEvent(title string, data interface{}, isError bool){
  var text string
  if isError {
    text = data.(error).Error()
  } else {
    text = data.(string)
  }
  event := datadog.NewEvent(title, text)
  if isError {
    event.AggregationKey = "error"
    event.AlertType = datadog.Error
    event.Tags = []string{"go-error"}
  }
  DoggoClient.Event(event)
}

func AddDoggoMetric(name string){
  DoggoClient.Incr(name, nil, 1)
}

func CountDoggoMetric(name string){
  // 1 second
  lock.Lock()
  counters[name] = counters[name] + 1
  lock.Unlock()
}

func DoggoSeive(){
  init:
  ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
  defer cancel()
  <- ctx.Done()
  // push all the counters
  lock.Lock()
  for name, val := range counters {
    DoggoClient.Count(name, val, nil, float64(val/1000))
    delete(counters, name)
  }
  lock.Unlock()
  goto init
}