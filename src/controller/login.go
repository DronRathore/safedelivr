package controller

import (
  "fmt"
  "config"
  "helpers"
  "models"
  "net/http"
  "crypto/sha256"
  "crypto/hmac"
  "encoding/hex"
  Time "time"
  "github.com/google/uuid"
  request "github.com/DronRathore/goexpress/request"
  response "github.com/DronRathore/goexpress/response"
  express "github.com/DronRathore/goexpress"
)

func LoginIndex(req *request.Request, res *response.Response, next func()){
  if res.Locals["logged_in"] != nil && res.Locals["logged_in"].(bool) == true {
    // already logged in
    res.JSON(map[string]interface{}{"success": true})
    return
  } else {
    var state = getUniqueString("")
    var url = config.Configuration.Github.Endpoint + "/authorize/?"
    url = url + "client_id=" + config.Configuration.Github.ClientId + "&"
    url = url + "redirect_uri=" + config.Configuration.Github.Success + "&"
    url = url + "scope=user&allow_signup=false&state=" + state
    // also add to the cookie
    var currTime = int64(Time.Now().Unix())
    res.Cookie.Add(&http.Cookie{
      Name: "transition",
      Value: state,
      Path: "/",
      Domain: "safedelivr.com",
      Expires: Time.Unix(currTime + 1860, 0)}) // 5 minutes to allow signin
    res.Redirect(url)
    return
  }
}

func LoginVerify(req *request.Request, res *response.Response, next func()){
  // check if already logged in
  if res.Locals["logged_in"] != nil && res.Locals["logged_in"].(bool) == true {
    res.JSON(map[string]string{"error": "Already logged in"})
    return
  }
  if len(req.Query["code"]) == 1 && len(req.Query["state"]) == 1 {
    var transitionCookie = req.Cookies.Get("transition")
    var state = req.Query["state"][0]
    var code = req.Query["code"][0]
    // validate transition cookie with state
    if state != transitionCookie {
      res.Header.SetStatus(403)
      res.Write("Action Forbidden").End()
      return
    }
    // else fetch the token and save it as a session
    var tokenStruct = helpers.GetGithubUserToken(code, state)
    if tokenStruct == nil {
      res.Header.SetStatus(400)
      res.Write("Bad Request").End()
      return
    }
    // we have the token, fetch user data and save both
    var userData = helpers.GetGithubUserData(tokenStruct)
    if userData == nil {
      res.Header.SetStatus(400)
      res.Write("Bad Request").End()
      return
    }
    // Create a new user or update the auth_token
    var user = &models.User{Email: userData.Email}
    exists, err := user.Exists()
    if err != nil {
      fmt.Println(err)
      res.Header.SetStatus(500)
      res.Write("Internal Error").End()
      return
    }

    if  exists == false {
      // create a new one
      done, err := user.Create(map[string]interface{}{
        "email": userData.Email,
        "name": userData.Name,
        "auth_token": userData.Access_Token,
        "company": userData.Company,
        "avatar_url": userData.Avatar_Url,
        "created_at": Time.Now(),
        "api_key": "",
        "user_type": 1,
        "location": userData.Location})
      // if no success, then throw error
      if done == false && err != nil {
        fmt.Println(err)
        res.Header.SetStatus(500)
        res.Write("Internal Error").End()
        return
      }
    } else {
      // update the auth token
      done, err := user.Update(map[string]interface{}{"auth_token": userData.Access_Token})
      if done == false && err != nil {
        // failed to save the user
        fmt.Println(err)
        res.Header.SetStatus(500)
        res.Write("Internal Error").End()
        return
      }
    }
    // User data fetched/Access Token updated
    userData.UUID = user.GetId()
    fmt.Println(user, user.GetId())
    var sessionKey = getUniqueString(tokenStruct.Access_Token)
    if helpers.SaveSession(sessionKey, userData) != true {
      // failed to save the user?
      res.Header.SetStatus(500)
      res.JSON(map[string]string{"error": "Failed to save user"})
      return
    }
    // add a session cookie
    var currTime = int64(Time.Now().Unix())
    res.Cookie.Add(&http.Cookie{
      Name: config.Configuration.Session.Cookie,
      Value: sessionKey,
      Path: "/",
      Expires: Time.Unix(currTime + 1860000, 0),
      Domain: "safedelivr.com"})
    
    // redirect to dashboard
    res.Redirect("/dashboard")
    return
  } else {
    res.Header.SetStatus(403)
    res.Write("Action Forbidden").End()
  }
}

func getUniqueString(key string) string {
  defer func(){
    if r := recover(); r != nil {
      // safe catch UUID failure
    }
  }()
  var hashKey []byte
  if key == "" {
    hashKey = []byte(config.Configuration.Github.ClientSecret)
  } else {
    hashKey = []byte(key)
  }
  var hash = hmac.New(sha256.New, hashKey)
  hash.Write([]byte(uuid.New().String()))
  return hex.EncodeToString(hash.Sum(nil))
}

var LoginController = func() interface{} {
  var Router = express.Router()
  Router.Get("/api/login", LoginIndex)
  Router.Get("/api/login/oAuth", LoginVerify)
  return *Router
}()