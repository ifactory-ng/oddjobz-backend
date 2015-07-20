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
	SubjectID string `json:"subjectid"`
	ObjectID  string `json:"objectid"`
	Review    Review `json:"review,omitempty"`
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
	log.Println(feed.Review)

	newReviewRedisScript := redis.NewScript(`
		local id = redis.call("incr", "posts:next_id")
		redis.call("set", "posts:"..id, ARGV[1])
		redis.call("lpush", "global:timeline", id)
		redis.call("lpush", "users:"..KEYS[1]..":timeline", id)
		local members = redis.call("smembers", "users:"..KEYS[1]..":followers")

		for i=1,#members do
			redis.call("lpush", "users:"..members[i]..":timeline", id)
		end
		return 1
	`)

	x, err := json.Marshal(feed)

	if err != nil {
		log.Println("error:", err)
	}

	_, err = newReviewRedisScript.Run(c.redis, []string{feed.SubjectID}, []string{string(x)}).Result()
	if err != nil {
		log.Println(err)
	}
	//log.Println(feed)

}

func (c *appContext) userFeedsHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := userget(r)

	FeedsRedisScript := redis.NewScript(`
		local ids = redis.call("lrange", "users:"..KEYS[1]..":timeline", 0, 9)
		-- print(unpack(ids))

		local nlist = {}

		for i=1,#ids do
			nlist[i] = "posts:"..ids[i]
		end

		local x = redis.call("mget", unpack(nlist))

		return x
	`)
	log.Println(user.Username)
	resp, err := FeedsRedisScript.Run(c.redis, []string{user.Username}, []string{"fffg"}).Result()
	if err != nil {
		log.Println(resp, err)

	}
	var results []Feed

	loop := resp.([]interface{})

	for _, rr := range loop {

		x := Feed{}

		err = json.Unmarshal([]byte(rr.(string)), &x)
		if err != nil {
			log.Println(err)
		}

		results = append(results, x)
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")

	err = json.NewEncoder(w).Encode(FeedsCollection{results})
	if err != nil {
		log.Println(err)
	}

}
