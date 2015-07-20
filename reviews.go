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

//types

//Review struct carries data fr each reviews
type Review struct {
	ID        bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
	Username  string        `json:"username"`
	SkillSlug string        `json:"skillslug"`
	Review    string        `json:"review"`
	Rating    int           `json:"rating"`
	User      User          `json:"user,omitempty" bson:"omitempty"`
}

//ReviewsCollection can carry a slice of reviews, and follws a jsn api standard when serialized
type ReviewsCollection struct {
	Data []Review `json:"data"`
}

//ReviewResource is like ReviewsCollection for single data
type ReviewResource struct {
	Data Review `json:"data"`
}

//ReviewRepo is basically contains a mng collection, and s being a struct culd have methds that act on the data in the collection
type ReviewRepo struct {
	coll *mgo.Collection
}

//Utility methods

//All returns all reviews tied t a skill of a particular slugname
func (r *ReviewRepo) All(query string) (ReviewsCollection, error) {
	result := ReviewsCollection{[]Review{}}
	err := r.coll.Find(bson.M{
		"skillslug": query,
	}).All(&result.Data) //TODO: Add pagination
	if err != nil {
		return result, err
	}
	return result, nil
}

//Create is methd that creates a review, while at the same time adding it as a feed
func (r *ReviewRepo) Create(review *Review) error {
	id := bson.NewObjectId()

	_, err := r.coll.UpsertId(id, review)
	if err != nil {
		return err
	}

	review.ID = id

	return nil
}

//Handlers

func (c *appContext) reviewsHandler(w http.ResponseWriter, r *http.Request) {
	repo := ReviewRepo{c.db.C("reviews")}
	params := context.Get(r, "params").(httprouter.Params)
	skillslug := params.ByName("slug")
	reviews, err := repo.All(skillslug)
	if err != nil {
		log.Println(err)
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")
	json.NewEncoder(w).Encode(reviews)

}

func (c *appContext) newReviewHandler(w http.ResponseWriter, r *http.Request) {

	user, err := userget(r)
	if err != nil {
		log.Println(err)
	}

	repo := ReviewRepo{c.db.C("reviews")}
	params := context.Get(r, "params").(httprouter.Params)
	skillslug := params.ByName("slug")
	body := context.Get(r, "body").(*ReviewResource)
	log.Println(skillslug)

	body.Data.SkillSlug = skillslug
	body.Data.Username = user.Username
	err = repo.Create(&body.Data)
	if err != nil {
		log.Println(err)
	}

	c.newReviewFeed(&body.Data)
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(body)
}
