package main

import (
	"math/rand"
	"net/http"

	"github.com/gorilla/context"
	"github.com/mitchellh/mapstructure"
)

func randSeq(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]

	}
	return string(b)

}

func userget(r *http.Request) (User, error) {
	u := context.Get(r, "User")
	var user User
	err := mapstructure.Decode(u, &user)
	if err != nil {
		return user, err
	}
	return user, nil

}
