package main

import (
	"encoding/json"
	"log"
	"net/http"

	"gopkg.in/redis.v2"
)

//Feed helps me serialize feed data to store in redis
type Feed struct {
	Type      string `json:"type"`
	Subject   string `json:"subject"`
	Object    string `json:"object"`
	SubjectID string `json:"objectid"`
	ObjectID  string `json:"objectid"`
	Review    Review `json:"review"`
}

//FeedsCollection  heelps me store and retrieve lists and send a schema compliant list
type FeedsCollection struct {
	Data []Feed `json:"data"`
}

//Utility Methds

func (c *appContext) newReviewFeed(review *Review) {
	feed := Feed{}
	feed.Type = "review"
	feed.ObjectID = review.SkillSlug
	feed.SubjectID = review.Username
	feed.Review = *review

	newReviewRedisScript := redis.NewScript(`
		local id = redis.call("incr", "posts:next_id")
		redis.call("set", "posts:"..id, ARGV[1])
		redis.call("lpush", "global:timeline", id)
		redis.call("lpush", "users:"..KEYS[1].."timeline", id)
		local username = KEYS[1]
		local members = redis.call("smembers", "users:"..username..":followers")

		for i=1,#members do
			redis.call("lpush", "users:"..i..":timeline", id)
		end
		return 1
	`)

	x, err := json.Marshal(feed)
	if err != nil {
		log.Println("error:", err)

	}

	resp, err := newReviewRedisScript.Run(c.redis, []string{feed.SubjectID}, []string{string(x)}).Result()
	log.Println(resp, err)
	if err != nil {
		log.Println(err)
	}
	log.Println(feed)

}

func (c *appContext) userFeedsHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := userget(r)

	FeedsRedisScript := redis.NewScript(`
		local ids = redis.call("lrange", "users:"..KEYS[1]..":timeline", 0, 9)
		-- local x = redis.call("mget", ids)
		return ids
	`)
	log.Println(user.Username)
	resp, err := FeedsRedisScript.Run(c.redis, []string{user.Username}, []string{"fffg"}).Result()
	log.Println(resp, err)

}
