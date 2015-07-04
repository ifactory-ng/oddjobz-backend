package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Main handlers
func (c *appContext) authHandler(w http.ResponseWriter, r *http.Request) {
	u := User{}
	user := &User{}
	log.Println(r.Body)
	err := json.NewDecoder(r.Body).Decode(&u)
	log.Println(u)

	if u.Provider == "local" {
		if u.Name != "" {
			phash, err := bcrypt.GenerateFromPassword([]byte(u.Password), Cost)
			if err != nil {
				log.Println(err)
			}

			C := c.db.C("users")

			u.PID = bson.NewObjectId().Hex()

			change := mgo.Change{
				Update: bson.M{
					"$set": bson.M{
						"pid":      u.PID,
						"username": u.Username,
						"name":     u.Name,
						"email":    u.Email,
						"password": phash,
					},
				},
				Upsert:    true,
				ReturnNew: true,
			}

			_, err = C.Find(bson.M{"email": u.Email, "provider": u.Provider}).Apply(change, &u)
			//log.Println(info)

			if err != nil {
				log.Println(err)
			}

			user = &u

		} else {
			xx := c.db.C("users")
			var res User
			err := xx.Find(bson.M{
				"provider": "local",
				"email":    u.Email,
			}).One(&res)
			if err != nil {
				log.Println(err)
			}

			err = bcrypt.CompareHashAndPassword([]byte(res.Password), []byte(u.Password))
			if err != nil {
				log.Println("password err")
				log.Println(err)
			}
			user = &res
		}
	} else {
		user, err = c.Authenticate(&u, u.Provider)

		if err != nil {
			log.Println(err)
		}

	}

	if user.Username == "" {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)

		response := struct {
			User    *User  `json:"user"`
			Message string `json:"message"`
			Token   string `json:"token"`
		}{
			User:    user,
			Message: "Sign up not yet complete",
		}

		json.NewEncoder(w).Encode(response)

	} else {
		// create a signer for rsa 256
		t := jwt.New(jwt.GetSigningMethod("RS256"))

		// set our claims
		t.Claims["AccessToken"] = user.Permission
		t.Claims["User"] = user

		log.Println("the tokened user is", user)
		// set the expire time
		// see http://tools.ietf.org/html/draft-ietf-oauth-json-web-token-20#section-4.1.4
		t.Claims["exp"] = time.Now().Add(time.Minute * 3000).Unix()
		tokenString, err := t.SignedString(c.signKey)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "Sorry, error while Signing Token!")
			log.Printf("Token Signing error: %v\n", err)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:       c.token,
			Value:      tokenString,
			Path:       "/",
			RawExpires: "0",
		})

		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)

		response := struct {
			User    *User  `json:"user"`
			Message string `json:"message"`
			Token   string `json:"token"`
		}{
			User:    user,
			Message: "SDasd",
			Token:   tokenString,
		}
		//log.Println(response)
		json.NewEncoder(w).Encode(response)
	}
}
