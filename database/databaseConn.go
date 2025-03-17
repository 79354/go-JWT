package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func DBinstance() *mongo.Client{
	// Loads the environment variables
	err := godotenv.Load(".env")
	if err != nil{
		log.Fatal("Error loading the .env file")
	}

	// MongoDb conn string from env variables
	MongoDb := os.Getenv("MONGODB_URL")

	// create context with Timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(MongoDb))
	if err != nil{
		log.Fatal(err)
	}

	// Test the connection
	err = client.Ping(ctx, nil)
	if err != nil{
		log.Fatal(err)
	}

	fmt.Println("Connected to MongoDb!")
	
	return client
}

var Client *mongo.Client = DBinstance()

func OpenCollection(client *mongo.Client, collectionName string) *mongo.Collection{
	// database managed in cloud: cluster0
	var collection *mongo.Collection = client.Database("cluster0").Collection(collectionName)
	return collection
}