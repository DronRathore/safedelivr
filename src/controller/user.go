package controller

import (
  "models"
  "io/ioutil"
  "encoding/json"
  "fmt"
  "doggo"
  "config"
  "redis"
  "net/http"
  "time"
  request "github.com/DronRathore/goexpress/request"
  response "github.com/DronRathore/goexpress/response"
  express "github.com/DronRathore/goexpress"
)

func IsLoggedIn(res *response.Response) bool {
  if res.Locals["logged_in"] != nil && res.Locals["user"] != nil && res.Locals["logged_in"].(bool) == true {
    return true
  }
  return false
}
func UserIndex(req *request.Request, res *response.Response, next func()){
  if IsLoggedIn(res) == true {
    var userData = res.Locals["user"].(*models.UserRedis)
    if userData == nil {
      goto Send403
    }
  	res.JSON(map[string]interface{}{
      "success": true,
      "email": userData.Email,
      "name": userData.Name,
      "avatar_url": userData.Avatar_Url,
      "company": userData.Company,
      "uuid": userData.UUID,
      "api_key": userData.Api_key,
      "location": userData.Location})
    return
  }
  Send403:
    res.Header.SetStatus(403)
    res.JSON(map[string]string{"error": "Please Login first"})
}

func UserUpdate(req *request.Request, res *response.Response, next func()) {
  if IsLoggedIn(res) == true {
    var userData = res.Locals["user"].(*models.UserRedis)
    if userData == nil {
      goto Send403
    }
    var user = models.User{Email:userData.Email}
    exists, err := user.Exists()
    if err != nil {
      fmt.Println(err)
      goto Send500
    }
    if exists == false {
      goto Send403
    }
    // read the json post
    data, err := ioutil.ReadAll(req.GetRaw().Body)
    if err == nil {
      var updateObj = make(map[string]interface{})
      err := json.Unmarshal(data, &updateObj)
      if err != nil {
        goto Send500
      } else {
        // update and populate the Object
        // avoid primary key from updation
        // we are like spotify, won't let you change email
        delete(updateObj, "email")
        delete(updateObj, "user_id")
        done, _ := user.Update(updateObj)
        if done == true {
          res.JSON(updateObj)
          return
        } else {
          goto Send500
        }
      }
    } else {
      goto Send500
    }
  }
  Send403:
    doggo.AddDoggoMetric("user.login.fail")
    res.Header.SetStatus(403)
    res.JSON(map[string]string{"error": "Please Login first"})
    return
  Send500:
    res.Header.SetStatus(500)
    res.JSON(map[string]string{"error": "Internal Server Error"})
}
/*
  Logout handler
*/
func UserLogout(req *request.Request, res *response.Response, next func()){
  if IsLoggedIn(res) {
    redis.Redis.Del(req.Cookies.Get(config.Configuration.Session.Cookie))
    res.Cookie.Add(&http.Cookie{
      Name: config.Configuration.Session.Cookie,
      Value: "",
      Path: "/",
      Domain: "safedelivr.com",
      Expires: time.Unix(int64(time.Now().Unix()) - 186000, 0)})
    res.Redirect("/")
    res.End()
  } else {
    res.Redirect("/")
    res.End()
  }
}

var UserController = func() interface{} {
  var Router = express.Router()
  Router.Get("/api/user", UserIndex)
  Router.Get("/api/user/logout", UserLogout)
  Router.Post("/api/user", UserUpdate)
  return *Router
}()
