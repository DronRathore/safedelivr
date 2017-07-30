package router

import (
	// "config"
	"middleware"
	"controller"
  express "github.com/DronRathore/goexpress"
)

func SetRoutes(Express express.ExpressInterface){
  Express.Use(middleware.CheckSession)
  Express.Use(controller.LoginController)
  Express.Use(controller.UserController)
  Express.Use(controller.BatchController)
}