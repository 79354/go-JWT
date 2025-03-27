package controllers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
	
	"go-jwt/database"
	"go-jwt/helpers"
	"go-jwt/models"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	userCollection  *mongo.Collection = database.OpenCollection(database.Client, "user")
	validate		*validator.Validate = validator.New() // to check if required feilds are missing or invalid
)


func HashPassword(password string) string{
	bytes, err:= bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil{
		log.Panic(err)
		// return "", fmt.Errorf("error hashing password: %v", err)
	}
	return string(bytes)
}

func VerifyPassword(userPassword string, providedPassword string) (bool, string){
	err := bcrypt.CompareHashAndPassword([]byte(providedPassword), []byte(userPassword))
	check := true
	msg := ""
	
	if err != nil{
		msg = fmt.Sprintf("email or password is incorrect")
		check = false
	}
	return check, msg
}

// user credentials (JSON) -> struct -> store the data from struct (MongoDB)
func Signup()gin.HandlerFunc{

	return func(c *gin.Context){
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var user models.User

		if err := c.BindJSON(&user); err != nil{
			// JSON serializes the given struct as JSON into the response body. It also sets the Content-Type as "application/json".
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// validator checks for missing and required feilds
		if err := validate.Struct(user); err != nil{
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		hashesPassword := HashPassword(*user.Password) // users bcrypt.GenerateFromPassword(), sercurely hash the pass before storing in database
		user.Password = &hashedPassword

		// Checks if email already exists
		emailCount, err := user.Collection.CountDocuments(ctx, bson.M{"email": user.Email})

		if err != nil{
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while checking for the email"})
			return
		}
		if emailCount > 1{
			c.JSON(http.StatusConflict, gin.H{"error": "email already exists"})
		}

		// Checks if phone number already exists
		phoneCount, err = userCollection.CountDocuments(ctx, bson.D{"email": user.Phone})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while checking for the phone"})
			return
		}
		
		if phoneCount > 1{
			c.JSON(http.StatusConflict, gin.H{"error": "error number already exists"})
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

		result, insertErr := userCollection.InsertOne(ctx, user)
		if insertErr != nil{
			msg := fmt.Sprintf("User item wasn't created")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		c.JSON(http.StatusCreated, result)
	}
}

func login() gin.HandlerFunc{
	return func(c gin.Context){
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var user, foundUser models.User

		if err := c.BindJSON(&user); err != nil{
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Find user email
		err := userCollection.FindOne(ctx, bson.M{"email": user.Email}).Decode(&foundUser)
		if err != nil{
			c.JSON(http.StatusUnauthorized, gin.H{"error": "email or password is incorrect"})
			return
		}

		// verify Password
		passwordIsValid, msg := VerifyPassword(*user.Password, *foundUser.Password)
		if passwordIsValid != true{
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		token, refreshToken, err := helper.GenerateAllTokens(
			*foundUser.Email,
			*foundUser.FirstName,
			*foundUser.LastName,
			*foundUser.UserType,
			foundUser.UserId,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error generating tokens"})
			return
		}

		helpers.UpdateAllTokens(token, refreshToken, foundUser.UserId)

		// Retrieve Update User
		var updatedUser models.User
		err = userCollection.FindOne(ctx, bson.M{"user_id":foundUser.UserId}).Decode(&updatedUser)
		if err != nil{
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error retrieving user"})
			return
		}

		c.JSON(http.StatusOk, updatedUser)
	}
}

// This func retrieves a paginated list of users from the MongoDB and ensures that only admin users can access it
func GetUsers() gin.HandlerFunc{
	return func(c *gin.Context){
		// Enforce admin access
		if err := helpers.CheckUserType(c, "ADMIN"); err != nil{
			c.JSON(http.StatusForbidden, gin.h{"error": err.Error()})
			return
		}

		var ctx, cancel = context.WithTimeout(context.Background(), 10 * time.Second)
		defer cancel()

		// pagination parameters
		recordPerPage, _ := strconv.Atoi(c.DefaultQuery("recordPerPage", "10"))
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		
		// calculate the starting index
		startIndex := (page-1) *recordPerPage

		// MongoDB Aggregation pipeline
		// 1. Match docs	2. Group them	3. project only required feilds and pagination
		pipeline := mongo.Pipeline{
			{{"$match", bson.D{}}},
			{{"$group", bson.D{
				{"_id", nil},
				{"total_count", bson.D{{"$sum", 1}}},
				{"data", bson.D{{"$push", "$$ROOT"}}},
			}}},
			{{"$project", bson.D{
				{"_id", 0},
				{"total_count", 1},
				{"user_items", bson.D{{"$slice", []interface{}{"$data", startIndex, recordPerPage}}}},
			}}},
		}

		// Execute Aggression
		cursor, err := userCollection.Aggregate(ctx, pipeline)
		if err != nil{
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error listing the users"})
			return
		}

		var results []bson.M
		if err = cursor.All(ctx, &results); err != nil{
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error while processing user list"})
			return
		}

		c.JSON(http.StatusOK, results[0])
	}
}

func GetUser() gin.HandlerFunc{
	return func(c *gin.Context){
		userId := c.Param("user_id")

		if err := helpers.MatchUserTypeToUid(c, userId); err != nil{
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var user models.User
		err := userCollection.FindOne(ctx, bson.M{"user_id": userId}).Decode(&user)
		if err != nil{
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		c.JSON(http.StatusOK, user)
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