package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"gopkg.in/redis.v2"
)

//types

// Transaction struct holds information about each users skills, aids in marshalling to json and storing on the database
type Transaction struct {
	ID         bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
	Date       time.Time     `json:"date"`
	Type       string
	Amount     int
	AmountType string
	SubjectID  string
	ObjectID   string
}

//TransactionsCollection holds a slice of Skill structs within a Data key, to conform with the json api schema spec
type TransactionsCollection struct {
	Data []Transaction `json:"data"`
}

//TransactionRepo a mongo Collection that could get passed around
type TransactionRepo struct {
	coll *mgo.Collection
}

//Utility methods

//All returns all skills tied to a particular user, based on the username or ID of the user
func (r *TransactionRepo) All(query string) (TransactionsCollection, error) {
	//log.Println(query)
	result := TransactionsCollection{[]Transaction{}}
	err := r.coll.Find(bson.M{
		"$or": bson.M{
			"subject": query,
			"object":  query,
		}}).All(&result.Data)
	if err != nil {
		return result, err
	}

	return result, nil
}

//Create adds a transaction to the database
func (r *TransactionRepo) Create(transaction *Transaction) error {
	id := bson.NewObjectId()

	_, err := r.coll.UpsertId(id, transaction)
	if err != nil {
		return err
	}

	transaction.ID = id

	return nil
}

func (c *appContext) deductCredits(username string, credits string) error {

	deductCreditsRedisScript := redis.NewScript(`
		local credits = redis.call("get", "users:"..KEYS[1]..":credits")
		print(ARG[1])

		if credits>=ARGV[1] do
			redis.call("incrby", "-"..ARGV[1])
			return 1
		end

		return 0
	`)
	resp, err := deductCreditsRedisScript.Run(c.redis, []string{username}, []string{credits}).Result()
	if err != nil {
		log.Println(resp, err)
	}

	if resp != 1 {
		return errors.New("insufficient credits")
	}
	return nil

}

func (c *appContext) addCredits(username string, credits string) error {
	creds, err := strconv.Atoi(credits)
	if err != nil {
		log.Println("error:", err)
	}
	_, err = c.redis.IncrBy("users:"+username+":credits", int64(creds)).Result()
	if err != nil {
		return err
	}
	return nil

}

//Handlers

func (c *appContext) transactionsHandler(w http.ResponseWriter, r *http.Request) {
	repo := TransactionRepo{c.db.C("skills")}
	user, _ := userget(r)
	log.Println(user)
	skills, err := repo.All(user.Username)
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")
	json.NewEncoder(w).Encode(skills)
}
