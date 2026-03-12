package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/adi041518/Core1/util"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var DB *mongo.Database

/*
* connection with dbClient uri
* connect clinet with the db
* check cient is active or not
 */
func ConnectDB() *mongo.Database {
	uri := os.Getenv("MONGO_URI")
	dbName := os.Getenv("DB_NAME")

	clientOptions := options.Client().ApplyURI(uri)

	client, err := mongo.NewClient(clientOptions)
	if err != nil {
		log.Fatal("Error creating Mongo client:", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		log.Fatal("Error connecting to MongoDB:", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("MongoDB ping failed:", err)
	}

	fmt.Println("Connected to MongoDB Atlas!")

	DB = client.Database(dbName)
	return DB
}

/*
* Get collection and pass the collection
 */
func OpenCollections(collectionName string) *mongo.Collection {
	return DB.Collection(collectionName)
}

/*
* Input parameters:context,collection,document
* Which insert into particular collection
* Return count,error
 */
func CreateOne(c context.Context, collection *mongo.Collection, document interface{}) (*mongo.InsertOneResult, error) {
	res, err := collection.InsertOne(c, document)
	if err != nil {

		log.Println("Error while inserting the document", err)
		return nil, errors.New(util.ERR_WHILE_INSERTING)
	}
	return res, nil
}

/*
* Create Many records with the collection given
 */
func CreateMany(c context.Context, collection *mongo.Collection, documents []interface{}) (*mongo.InsertManyResult, error) {
	res, err := collection.InsertMany(c, documents)
	if err != nil {
		log.Println("Error while inserting the document", err)
		return nil, errors.New(util.ERR_WHILE_INSERTING_MANY)
	}
	return res, nil
}

/*
* To find the document inside particular db collection
* Check for it if error doesnot occur pass the variable data to it
* if err occur either no document found nor the findone error
* Return error
 */
func FindOne(ctx context.Context, collection *mongo.Collection, filter interface{}, result interface{}) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	SingleResult := collection.FindOne(ctxTimeout, filter)
	if err := SingleResult.Err(); err != nil {
		return err
	}
	if err := SingleResult.Decode(result); err != nil {
		log.Println("Error decoding document:", err)
		return err
	}
	// log.Println(result)
	return nil
}

/*
* To findAll inside the particular db collection
* Pass each document into the list of interface
* Check if the document present or not and then Decode and pass to the results
 */

func FindAll(c context.Context, collection *mongo.Collection, filter interface{}, opts *options.FindOptions) ([]interface{}, error) {
	if filter == nil {
		filter = bson.M{}
	}
	cursor, err := collection.Find(c, filter, opts)
	if err != nil {
		return nil, err
	}

	defer cursor.Close(c)

	var results []interface{}

	for cursor.Next(c) {
		doc := make(map[string]interface{})
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		results = append(results, doc)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

/*
* Find list of documents based on the filter and as well as the collection provided
* Based on the skip conditions we get them
 */
func FindByPage(c context.Context, collection *mongo.Collection, filter interface{}, page int, size int) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	skip := (page - 1) * size
	opts := options.Find()
	opts.SetSkip(int64(skip))
	opts.SetLimit(int64(size))
	opts.SetSort(bson.D{{Key: "UpdatedAt", Value: -1}})
	cursor, err := collection.Find(c, filter, opts)
	if cursor.Next(c) {
		doc := make(map[string]interface{})
		if err := cursor.Decode(&doc); err != nil {
			log.Println("Error decoding document:", err)
			return nil, err
		}
		results = append(results, doc)
	}
	if err != nil {
		return nil, err
	}
	return results, nil
}

/*
* Delete the particular document for the given collection filter condition
 */
func DeleteOne(c context.Context, collection *mongo.Collection, filter interface{}) (*mongo.DeleteResult, error) {
	count, err := collection.DeleteOne(c, filter)
	if err != nil {
		log.Println("Error while deleting the document")
		return nil, errors.New(util.ERR_WHILE_DELETING)
	}
	return count, nil
}

/*
* Delete documents in the particular collection based on the filter provided
 */
func DeleteMany(ctx context.Context, collection *mongo.Collection, filter interface{}) (*mongo.DeleteResult, error) {
	count, err := collection.DeleteMany(ctx, filter)
	if err != nil {
		log.Println("Error while deleting the document")
		return nil, errors.New(util.ERR_WHILE_DELETING)
	}
	return count, nil
}

/*
* Update the document based on the collection given
* Return error if the update fail
* Else return the updated count
 */
func UpdateOne(ctx context.Context, collection *mongo.Collection, filter interface{}, update interface{}) (*mongo.UpdateResult, error) {
	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Println("MongoDB UpdateOne error:", err)
		return nil, errors.New(util.ERR_WHILE_UPDATING)
	}
	return result, nil
}

/*
* Update the document based on the collection given
* Return error if the update fail
* Else return the updated counts
 */
func UpdateMany(ctx context.Context, collection *mongo.Collection, filter interface{}, update interface{}, opts *options.UpdateOptions) (*mongo.UpdateResult, error) {
	count, err := collection.UpdateMany(ctx, filter, update, opts)
	if err != nil {
		log.Println("Error while updating the documents")
		return nil, errors.New(util.ERR_WHILE_UPDATING)
	}
	return count, nil
}
