package middleware

import (
  "config"
  "redis"
  "helpers"
  request "github.com/DronRathore/goexpress/request"
  response "github.com/DronRathore/goexpress/response"
)

func CheckSession(req *request.Request, res *response.Response, next func()){
  var cookie = req.Cookies.Get(config.Configuration.Session.Cookie)
  if cookie == "" {
    res.Locals["logged_in"] = false
  } else {
    // check redis store if session still valid
    User, err := helpers.GetSession(cookie)
    if err != nil {
      res.Locals["logged_in"] = false
      redis.Redis.Del(cookie)
      res.Cookie.Del(config.Configuration.Session.Cookie)
    } else {
      // parse and add the user data
      res.Locals["user"] = User
      res.Locals["logged_in"] = true
    }
  }
}