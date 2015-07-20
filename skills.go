package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/extemporalgenome/slug"
	"github.com/gorilla/context"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//types

// Skill struct holds information about each users skills, aids in marshalling to json and storing on the database
type Skill struct {
	ID           bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
	Featured     int           `json:"featured,omitempty"`
	Slug         string        `json:"slug"`
	Name         string        `json:"name"`
	Summary      string        `json:"summary"`
	About        string        `json:"about"`
	Address      string        `json:"address"`
	City         string        `json:"city"`
	State        string        `json:"state"`
	Phone        string        `json:"phone"`
	Owner        string        `json:"owner"`
	Timestamp    time.Time     `json:"timestamp"`
	Images       []Images      `json:"images"`
	Rating       int           `json:"rating"`
	TotalReviews int           `json:"-"`
	ReviewsCount int           `json:"-"`
}

//SkillsCollection holds a slice of Skill structs within a Data key, to conform with the json api schema spec
type SkillsCollection struct {
	Data []Skill `json:"data"`
}

//SkillResource acts like SkillResource but carries information about a single skill
type SkillResource struct {
	Data Skill `json:"data"`
}

//SkillRepo a mongo Collection that could get passed around
type SkillRepo struct {
	coll *mgo.Collection
}

//Utility methods

//All returns all skills tied to a particular user, based on the username or ID of the user
func (r *SkillRepo) All(query string) (SkillsCollection, error) {
	//log.Println(query)
	result := SkillsCollection{[]Skill{}}
	err := r.coll.Find(bson.M{
		"owner": query,
	}).All(&result.Data)
	if err != nil {
		return result, err
	}

	return result, nil
}

//Find returns a SkillResource which contains a single skill
func (r *SkillRepo) Find(query string) (SkillResource, error) {
	result := SkillResource{}

	err := r.coll.Find(bson.M{
		"slug": query,
	}).One(&result.Data)
	if err != nil {
		return result, err
	}

	return result, nil
}

//Create adds a skill to the database, based on it's owner
func (r *SkillRepo) Create(skill *Skill) error {
	id := bson.NewObjectId()

	Slug := slug.Slug(skill.Name + " " + skill.City + " " + randSeq(7))
	skill.Slug = Slug
	_, err := r.coll.UpsertId(id, skill)
	if err != nil {
		return err
	}

	skill.ID = id

	return nil
}

//Update updates information about a skill
func (r *SkillRepo) Update(skill *Skill) error {
	log.Println(skill)
	err := r.coll.Update(
		bson.M{
			"slug": skill.Slug,
		}, skill)
	if err != nil {
		return err
	}

	return nil
}

//Delete removes a skill from the database
func (r *SkillRepo) Delete(id string) error {
	err := r.coll.RemoveId(bson.ObjectIdHex(id))
	if err != nil {
		return err
	}

	return nil
}

//Handlers
func (c *appContext) skillsHandler(w http.ResponseWriter, r *http.Request) {
	repo := SkillRepo{c.db.C("skills")}
	user, _ := userget(r)
	log.Println(user)
	skills, err := repo.All(user.Username)
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")
	json.NewEncoder(w).Encode(skills)
}

func (c *appContext) skillHandler(w http.ResponseWriter, r *http.Request) {
	params := context.Get(r, "params").(httprouter.Params)
	repo := SkillRepo{c.db.C("skills")}
	skill, err := repo.Find(params.ByName("slug"))
	if err != nil {
		panic(err)
	}
	skill.Data.Phone = ""
	skill.Data.Address = "hidden"

	w.Header().Set("Content-Type", "application/vnd.api+json")
	json.NewEncoder(w).Encode(skill)
}

func (c *appContext) createSkillHandler(w http.ResponseWriter, r *http.Request) {
	body := context.Get(r, "body").(*SkillResource)
	repo := SkillRepo{c.db.C("skills")}
	err := repo.Create(&body.Data)
	if err != nil {
		log.Println(err)
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(body)
}

func (c *appContext) updateSkillHandler(w http.ResponseWriter, r *http.Request) {
	params := context.Get(r, "params").(httprouter.Params)
	body := context.Get(r, "body").(*SkillResource)
	body.Data.Slug = params.ByName("slug")
	repo := SkillRepo{c.db.C("skills")}
	err := repo.Update(&body.Data)
	if err != nil {
		log.Println(err)
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(body)

}

func (c *appContext) deleteSkillHandler(w http.ResponseWriter, r *http.Request) {
	params := context.Get(r, "params").(httprouter.Params)
	repo := SkillRepo{c.db.C("skills")}
	err := repo.Delete(params.ByName("slug"))
	if err != nil {
		panic(err)
	}

	w.WriteHeader(http.StatusNoContent)
	w.Write([]byte("\n"))
}

func (c *appContext) getSkillContact(w http.ResponseWriter, r *http.Request) {
	params := context.Get(r, "params").(httprouter.Params)
	repo := SkillRepo{c.db.C("skills")}
	skill, err := repo.Find(params.ByName("slug"))
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")
	json.NewEncoder(w).Encode(skill)

}
