package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/context"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//User carries user data for exchange, especially in views
type User struct {
	ID         string `json:"id" bson:",omitempty"`
	PID        string `json:"pid" bson:",omitempty"`
	Provider   string `json:"provider"`
	Username   string `json:"username"`
	Password   string
	Permission string `json:"permission" bson:"permission,omitempty"`
	Image      string `json:"image"`
	Name       string `json:"name"`
	Link       string `json:"link"`
	Gender     string `json:"gender"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
}

//UsersCollection holds a slice of user structs under the key "data", which culd be marshalled and sent to a client under json schema standard
type UsersCollection struct {
	Data []User `json: "data"`
}

//UserResource acts like UsersCollection abve, but carries data abut a single user
type UserResource struct {
	Data User `json:"data"`
}

//UserRepo carries a mongo cllectiin fr which i could write  methods that can utilize
type UserRepo struct {
	coll *mgo.Collection
}

//Utility methods

//All returns all users in the cllection.
func (r *UserRepo) All() (UsersCollection, error) {
	result := UsersCollection{[]User{}}
	err := r.coll.Find(nil).All(&result.Data) //TODO: pagination
	if err != nil {
		return result, err
	}
	return result, nil
}

//Find wuld return a user struct basedon the username of the user, which is the query
func (r *UserRepo) Find(query string) (UserResource, error) {
	result := UserResource{}

	err := r.coll.Find(bson.M{
		"username": query,
	}).One(&result.Data)
	if err != nil {
		return result, err
	}
	return result, nil
}

//Authenticate check if user exists if not create a new user document NewUser function is called within this function. note the user struct being passed
//to this function should alredi contain a self generated objectid
func (c *appContext) Authenticate(user *User, provider string) (*User, error) {
	log.Println("Authenticate")
	result := User{}
	C := c.db.C("users")

	log.Println(user.PID)
	log.Println(provider)

	var change mgo.Change
	if user.Username != "" {
		err := c.redis.SAdd("users", user.Username).Err()
		if err != nil {
			log.Println(err)
		}
		change = mgo.Change{
			Update: bson.M{"$set": bson.M{
				"pid":      user.PID,
				"name":     user.Name,
				"email":    user.Email,
				"image":    user.Image,
				"username": user.Username,
				"phone":    user.Phone,
			},
			},
			Upsert:    true,
			ReturnNew: true,
		}
	} else {
		change = mgo.Change{
			Update: bson.M{"$set": bson.M{
				"pid":   user.PID,
				"name":  user.Name,
				"email": user.Email,
				"image": user.Image,
			},
			},
			Upsert:    true,
			ReturnNew: true,
		}
	}
	info, err := C.Find(bson.M{"pid": user.PID, "provider": provider}).Apply(change, &result)
	log.Println(info)
	log.Println(result)

	if err != nil {
		return &result, err
	}
	//if result.Provider != "" {
	//	return &result, nil
	//}

	//return c.NewUser(user, provider)
	return &result, nil
}

//NewUser is for adding a new user to the database. Please note that what you pass to the function is a pointer to the actual data, note the data its self. ie newUser(&NameofVariable)
func (c *appContext) NewUser(data *User) (*User, error) {

	collection := c.db.C("users")
	data.Provider = data.Provider

	err := collection.Insert(data)
	if err != nil {
		log.Println(err)
		return data, err
	}

	return data, nil
}

func (c *appContext) meHandler(w http.ResponseWriter, r *http.Request) {
	user, err := userget(r)
	if err != nil {
		log.Println(err)
	}

	//log.Println(user)
	repo := UserRepo{c.db.C("users")}
	userD, err := repo.Find(user.Username)
	if err != nil {
		log.Println(err)
	}

	w.Header().Set("Content-Type", "application/vdn.api+json")
	json.NewEncoder(w).Encode(userD)

}

func (c *appContext) userHandler(w http.ResponseWriter, r *http.Request) {
	params := context.Get(r, "params").(httprouter.Params)
	repo := UserRepo{c.db.C("users")}
	username := params.ByName("username")
	log.Println(username)
	user, err := repo.Find(username)
	if err != nil {
		log.Println(err)
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")
	json.NewEncoder(w).Encode(user)
}

func (c *appContext) followUserHandler(w http.ResponseWriter, r *http.Request) {
	params := context.Get(r, "params").(httprouter.Params)
	followUsername := params.ByName("username")
	user, err := userget(r)
	if err != nil {
		log.Println(err)
	}
	keyFollowing := "users:" + user.Username + ":following"
	keyFollowers := "users:" + followUsername + ":followers"

	err = c.redis.SAdd(keyFollowing, followUsername).Err()
	if err != nil {
		log.Println(err)
	}

	err = c.redis.SAdd(keyFollowers, user.Username).Err()
	if err != nil {
		log.Println(err)
	}

}
