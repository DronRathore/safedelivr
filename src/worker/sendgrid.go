package worker

import(
  "net/http"
  "strings"
  "helpers"
  "fmt"
  "time"
  "errors"
  "config"
  "encoding/json"
  "github.com/streadway/amqp"
)

var SendGridClient *http.Client

func InitSg(){
  transport := &http.Transport{
    MaxIdleConns:       200,
    IdleConnTimeout:    60 * time.Second,
    DisableCompression: false,
  }
  SendGridClient = &http.Client{Transport: transport}
}

func SGListener(packet amqp.Delivery){
  // the channel and routing keys are for failures
  // in which the request would be forwarded to another provider
  // as you pass it here
  fmt.Println("Sendgrid recieved a packet", packet.RoutingKey, string(packet.Body))
  parts := strings.Split(packet.RoutingKey, ".")
  if parts[0] == "log" {
    ProcessLog(packet, "sendgrid", GetSgEmailVars, SendSgRequest)
  } else {
    ProcessMail(packet, "sendgrid", GetSgEmailVars, SendSgRequest)
  }
}

/*
  Sends the request
  @return
    StatusCode int
    Nack Packet bool
    Error error
*/
func SendSgRequest(body *string) (int, bool, error) {
 request, err := getSGRequestObject(body)
   if err != nil {
    // negative ack packet and requeue it
    return -1, true, err
   }
   response, err := SendGridClient.Do(request)
   if err != nil {
    // todo: log error
    fmt.Println(err)
    // negative acknowledge packet and requeue the packet
    return -1, true, err
   }
   fmt.Println("Response Status", response.StatusCode)
   return response.StatusCode, false, nil
}

/*
  Returns a http.Request object for POST endpoint of Sendgrid
*/
func getSGRequestObject(data *string) (*http.Request, error) {
  request, err := http.NewRequest("POST", config.Configuration.Sendgrid.Endpoint, strings.NewReader(*data))
  if err != nil {
    // todo: log error
    return nil, err
  }
  request.Header.Add("authorization", "Bearer " + config.Configuration.Sendgrid.Key)
  request.Header.Add("Content-Type", "application/json")
  return request, nil
}

/*
  Creates a JSON body out of options passed to it
*/
func GetSgEmailVars(options map[string]string, custom_args map[string]string) (string, error) {
  requestBody := make(map[string]interface{})
  var toArray []string
  // var ccArray []string
  var isBulk = false
  if options["isBulk"] == "true" {
    isBulk = true
  }
  if options["to"] != "" && len(options["to"]) < 5 {
    // cannot process entity, push error
    return "", errors.New("Not a valid recievers list")
  }
  // split string
  toArray = strings.Split(options["to"], ",")
  personalizations := make([]map[string]interface{}, 0)
  // create separate personalization array values
  for _, element := range getSGPersonalizationArray(options["subject"], toArray, isBulk, custom_args) {
    personalizations = append(personalizations, element)
  }
  requestBody["personalizations"] = personalizations
  // from option
  requestBody["from"] = map[string]interface{}{
    "email": options["from"],
    "name": options["name"],
  }
  requestBody["reply_to"] = map[string]interface{}{
    "email": options["from"],
    "name": options["name"],
  }
  // content is an HTML content
  requestBody["content"] = []map[string]interface{}{
    map[string]interface{}{
      "type": "text/html",
      "value": options["content"],
      },
    }
  // todo: add cc, bcc: fields
  body, err := json.Marshal(requestBody)
  return string(body), err
}
/*
  Returns a Sendgrid Personalization Map array for email send request
  @params:
    subject: subject line of email
    toArray: An array of recipients
    isBulk: If the request was for bulk email
    custom_args: A list of custom args used for tracking in webhooks

  @return:
    The format of personalization map is:
    personalization: [
      {
        to: [{email: "email"}, ... next ],
        subject: string,
        custom_args: map
      },
      {
        to: [{email: "email"}, ...next], // in case of batch need seperate maps
        subject: string,
        custom_args: map
      }
    ]  
*/
func getSGPersonalizationArray(subject string, toArray []string, isBulk bool, custom_args map[string]string) ([]map[string]interface{}) {
  // define an empty map array
  var personalizations []map[string]interface{} = make([]map[string]interface{}, 0)
  // loop over all the recipients
  for _, email := range toArray {
    // strip all the space characters from email
    email = strings.Replace(email, " ", "", -1)
    // check if it is a valid email
    if helpers.IsEmail(email) == true {
      if isBulk {
        // keep appending new map objects
        var toMap = map[string]interface{}{
         "to": []map[string]string{map[string]string{"email": email}},
         "subject": string(subject),
         "custom_args": custom_args}
        personalizations = append(personalizations, toMap)
      } else {
        // append a new email object
        if len(personalizations) > 0 {
          personalizations[0]["to"] = append(personalizations[0]["to"].([]map[string]string), map[string]string{"email": email})
        } else {
          personalizations = append(personalizations, map[string]interface{}{
         "to": []map[string]string{map[string]string{"email": email}},
         "subject": string(subject),
         "custom_args": custom_args,
         })
        } // if len(personalization)
      } // if bulk
    } // if valid email
  } // for loop
  return personalizations
}