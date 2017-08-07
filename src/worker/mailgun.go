package worker

import(
  "net/http"
  "net/url"
  "strings"
  // "rabbit"
  "helpers"
  "fmt"
  "time"
  "errors"
  "config"
  "encoding/json"
  "io/ioutil"
  "github.com/streadway/amqp"
)

var MailGunClient *http.Client

func InitMg(){
  transport := &http.Transport{
    MaxIdleConns:       200,
    IdleConnTimeout:    60 * time.Second,
    DisableCompression: false,
  }
  MailGunClient = &http.Client{Transport: transport}
}

func MGListener(packet amqp.Delivery){
  // the channel and routing keys are for failures
  // in which the request would be forwarded to another provider
  // as you pass it here
  fmt.Println("Mailgun recieved a packet", packet.RoutingKey, string(packet.Body))
  parts := strings.Split(packet.RoutingKey, ".")
  if parts[0] == "log" {
    ProcessLog(packet, "mailgun", GetMgEmailVars, SendMgRequest)
  } else {
    ProcessMail(packet, "mailgun", GetMgEmailVars, SendMgRequest)
  }
}
/*
  Sends the request
  @return
    StatusCode int
    Nack Packet bool
    Error error
*/
func SendMgRequest(body *string) (int, bool, error) {
 request, err := getMGRequestObject(body)
   if err != nil {
    // negative ack packet and requeue it
    return -1, true, err
   }
   response, err := MailGunClient.Do(request)
   if err != nil {
    // todo: log error
    fmt.Println(err)
    // negative acknowledge packet and requeue the packet
    return -1, true, err
   }
   fmt.Println(response.StatusCode)
   data , _ := ioutil.ReadAll(response.Body)
   fmt.Println(string(data))
   return response.StatusCode, false, nil
}

/*
  Returns a http.Request object for POST endpoint of Sendgrid
*/
func getMGRequestObject(data *string) (*http.Request, error) {
  var url = config.Configuration.MailGun.Endpoint + "/" +  config.Configuration.MailGun.Domain + "/messages"
  request, err := http.NewRequest("POST", url, strings.NewReader(*data))
  if err != nil {
    // todo: log error
    return nil, err
  }
  request.SetBasicAuth("api", config.Configuration.MailGun.Key)
  request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
  return request, nil
}

/*
  Creates a JSON body out of options passed to it
*/
func GetMgEmailVars(options map[string]string, custom_args map[string]string) (string, error) {
  requestBody := url.Values{}
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
  // get the rc variable string
  var rcVariableString = getMgRecipientVariables(toArray, isBulk, custom_args)
  // from option
  var from = options["name"] + "<" + options["from"] + ">"
  requestBody.Add("from", from)
  // add all the recipients
  for _, email := range toArray {
    to := "<" + email + ">"
    requestBody.Add("to", to)
  }
  requestBody.Add("subject", options["subject"])
  requestBody.Add("html", options["content"])
  requestBody.Add("h:Reply-To", from)
  if isBulk == true {
    requestBody.Add("recipient-variables", rcVariableString)
  }
  // Add custom variables at the global level too
  for key, value := range custom_args {
    // convert underscore to hyphens
    key = strings.Replace(key, "_", "-", -1)
    requestBody.Add("v:" + key, value)
  }
  // Encode the request params
  return requestBody.Encode(), nil
}

/*
  Returns a Mailgun Recipient Variables Map array for email send request
  @params:
    toArray: An array of recipients
    isBulk: If the request was for bulk email
    custom_args: A list of custom args used for tracking in webhooks

  @return:
    The format of recipient-variables map is:
    {
      "email": {...custom vars...}
    }
*/
func getMgRecipientVariables(toArray []string, isBulk bool, custom_args map[string]string) (string) {
  // indivdual emails doesn't need rc-variables
  if isBulk == false {
    return ""
  }
  var rcVariables map[string]interface{} = make(map[string]interface{})
  // loop over all the recipients
  for _, email := range toArray {
    // strip all the space characters from email
    email = strings.Replace(email, " ", "", -1)
    // check if it is a valid email
    if helpers.IsEmail(email) == true {
      rcVariables[email] = custom_args
    } // if valid email
  } // for loop
  str, _ := json.Marshal(rcVariables)
  return string(str)
}