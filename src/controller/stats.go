package controller

import (
	request "github.com/DronRathore/goexpress/request"
  response "github.com/DronRathore/goexpress/response"
  express "github.com/DronRathore/goexpress"
  "models"
  "github.com/gocql/gocql"
)
/*
  Returns stats of last week
*/
func GetStats(req *request.Request, res *response.Response, next func()){
  if IsLoggedIn(res) == false {
    res.JSON(map[string]interface{}{"error": "Access Denied"})
    return
  }
  var userData = res.Locals["user"].(*models.UserRedis)
  user_id, _ := gocql.ParseUUID(userData.UUID)
  data := models.GetWeekStats(user_id)
  res.JSON(data)
}

var StatsController = func() interface{} {
  var Router = express.Router()
  Router.Get("/api/stats", GetStats)
  return *Router
}()