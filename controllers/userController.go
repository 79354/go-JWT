package controllers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	helper "go-jwt/helpers"
	"go-jwt/database"
	"go-jwt/models"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"
)

var UserCollection *mongo.Collection = database.OpenCollection(database.Client, "user")
var validate = validator.New() 

func HashPassword(){

}

func VerifyPassword(){

}

func Signup(){

}

func Login(){

}

func GetUsers(){

}

func GetUser() gin.HandlerFunc{
	return func(c *gin.Context{
		userId := c.Param("user_id")

		if err:= helper.MatchUserTypeToUid(c, userId); err != nil{
			c.JSON(http.StatusBadRequest, gin.H{"error":err.Error()})
		}
	}
}


// main -> routes -> database -> controller