package routes

import(
	controller "go-jwt/controllers"
	"github.com/gin-gonic/gin"
)

func AuthRoutes(routes *gin.Engine){
	routes.POST("users/signup", controller.Signup())
	routes.POST("users/login", controller.Login())
}