package controllers

import (
	"context"
	"fmt"
	"go-jwt/database"
	"go-jwt/helpers"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

var userCollection *mongo.Collection = database.OpenCollection(database.Client, "user")
var validate = validator.New()	// to check if required feilds are missing or invalid

func HashPassword(password string) string{
	bytes, err:= bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil{
		log.Panic(err)
	}
	return string(bytes)
}

func VerifyPassword(userPassword string, providedPassword string) (bool, string){
	err := bcrypt.CompareHashAndPassword([]byte(providedPassword), []byte(userPassword))
	check := true
	msg := ""
	
	if err != nil{
		msg = fmt.Sprintf("email of password is incorrect")
		check = false
	}
	return check, msg
}

// user credentials (JSON) -> struct -> store the data from struct (MongoDB)
func Signup()gin.HandlerFunc{

	return func(c *gin.Context){
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		var user models.User

		if err := c.BindJSON; err != nil{
			// JSON serializes the given struct as JSON into the response body. It also sets the Content-Type as "application/json".
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// validator checks for missing and required feilds
		validationErr := validate.Struct(user)
		if validationErr != nil{
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		password := HashPassword(*user.Password) // users bcrypt.GenerateFromPassword(), sercurely hash the pass before storing in database
		user.Password = &password

		count, err := user.Collection.CountDocuments(ctx, bson.M{"email": user.Email})
		defer cancel()	// releases resources associated with MongoDB query
		if err != nil{
			log.Panic(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while checking for the email"})
		}

		count, err = userCollection.CountDocuments(ctx, bson.M{"phone": user.Phone})
		defer cancel()
		if err != nil {
			log.Panic(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while checking for the phone"})
		}
		
		if count > 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error" : "this email or phone already exists"})
		}

		// Set Metadata feids
		user.CreatedAt, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		user.UpdatedAt, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		user.ID = primitive.NewObjectID()	// generates new unique mongodb Object ID
		user.UserId = user.ID.Hex()	// converts it to hex string for easy use

		// Generate JWT token, the tokens are stored in user record
		token, refreshToken, _ := helpers.GenerateAllTokens(*user.Email, *user.FirstName, *user.LastName, *user.UserType, *&user.UserId)
		user.Token = &token
		user.RefreshToken = &refreshToken

		resultInsertionNumber, insertErr := userCollection.InsertOne(ctx, user)
		if insertErr != nil{
			msg := fmt.Sprintf("User item wasn't created")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		}
		defer cancel()
		c.JSON(http.StatusOK, resultInsertionNumber)
	}
}

func login() gin.HandlerFunc{
	return func(c gin.Context){
		var ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		var user models.User
		var foundUser models.User

		if err := c.BindJSON(&user); err != nil{
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := userCollection.FindOne(ctx, bson.M{"email": user.Email}).Decode(&foundUser)
		defer cancel()
		if err != nil{
			c.JSON(http.StatusInternalServerError, gin.H{"error": "email or password is incorrect"})
		}

		passwordIsValid, msg := VerifyPassword(*user.Password, *foundUser.Password)
		if passwordIsValid != true{
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		if foundUser.Email == nil{
			c.JSON(http.StatusInternalServerError, gin.H{"error":"user not found"})
		}
		token, refreshToken, _ := helpers.GenerateAllTokens(*foundUser.Email, *foundUser.FirstName, *foundUser.LastName, *foundUser.UserType, *foundUser.UserId)
		helpers.UpdateAllTokens(token, refreshToken, foundUser.UserId)
		err = userCollection.FindOne(ctx, bson.M{"user_id":foundUser.UserId}).Decode(&foundUser)

		if err != nil{
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		c.JSON(http.StatusOk, foundUser)
	}
}

func GetUsers() gin.HandlerFunc{
	return func(c *gin.Context){
		if err := helpers.CheckUserType(c, "ADMIN"); err!=nil{
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		recordPerPage, err := strconv.Atoi(c.Query("recordPerPage"))
		if err != nil || recordPerPage < 1{
			recordPerPage = 10
		}
		page, err := strconv.Atoi(c.Query("page"))
		if err !=nil || page < 1{
			page = 1
		}

		startIndex := (page - 1) * recordPerPage
		startIndex, err = strconv.Atoi(c.Query("startIndex"))

		matchStage := bson.D{"$match": bson.D{}}
		matchStage :=
	}
}

/*
	Signup
	The Signup function is responsible for registering a new user. It:

	1. Receives JSON input from the client.
	2. Validates the input to ensure all required fields are provided.
	3. Checks for duplicate users (email and phone number).
	4. Hashes the password before storing it in the database.
	5. Generates JWT tokens for authentication.
	6. Saves the user in the MongoDB collection.
	7. Returns a response indicating success or failure.

	Extracts JSON from the request c.BindJSON(&user)
	validates user input validate.Struct(user)
	checks for existing email/ phone number (userCollection.CountDocuments)
	Hashes the password (bcrypt.GenerateFromPassword)
	Creates user metadata (timestamps, MongoDB ID)
	Generates JWT tokens (GenerateAllTokens)
	inserts the user into MongoDB (InsertOne)
	Returns success or error response c.JSON()
*/