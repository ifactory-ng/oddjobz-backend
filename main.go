package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/context"
	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"gopkg.in/mgo.v2"
	"gopkg.in/redis.v2"
)

type appContext struct {
	db        *mgo.Database
	verifyKey []byte
	signKey   []byte
	token     string

	domain string

	bucket *s3.Bucket
	redis  *redis.Client
}

const (
	//Cost is the well, cost of the bcrypt encryption used for storing user
	//passwords in the database
	Cost int = 5
)

// Router

// Router struct would carry the httprouter instance, so its methods could be verwritten and replaced with methds with wraphandler
type Router struct {
	*httprouter.Router
}

// Get is an endpoint to only accept requests of method GET
func (r *Router) Get(path string, handler http.Handler) {
	r.GET(path, wrapHandler(handler))
}

// Post is an endpoint to only accept requests of method POST
func (r *Router) Post(path string, handler http.Handler) {
	r.POST(path, wrapHandler(handler))
}

// Put is an endpoint to only accept requests of method PUT
func (r *Router) Put(path string, handler http.Handler) {
	r.PUT(path, wrapHandler(handler))
}

// Delete is an endpoint to only accept requests of method DELETE
func (r *Router) Delete(path string, handler http.Handler) {
	r.DELETE(path, wrapHandler(handler))
}

// NewRouter is a wrapper that makes the httprouter struct a child of the router struct
func NewRouter() *Router {
	return &Router{httprouter.New()}
}

func wrapHandler(h http.Handler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		context.Set(r, "params", ps)
		h.ServeHTTP(w, r)
	}
}

func checks() (REDISADDR, REDISPW, MONGOSERVER, MONGODB string, Public []byte, Private []byte, RootURL, AWSBucket string) {
	REDISADDR = os.Getenv("REDISURL")

	REDISPW = os.Getenv("REDISPW")

	if REDISADDR == "" {
		log.Println("No mongo server address set, resulting to default address")
		REDISADDR = "localhost:6379"

	}
	log.Println("REDISADDR is ", REDISADDR)

	MONGOSERVER = os.Getenv("MONGOLAB_URI")
	if MONGOSERVER == "" {
		log.Println("No mongo server address set, resulting to default address")
		MONGOSERVER = "localhost"
	}
	log.Println("MONGOSERVER is ", MONGOSERVER)

	MONGODB = os.Getenv("MONGODB")
	if MONGODB == "" {
		log.Println("No Mongo database name set, resulting to default")
		MONGODB = "oddjobz"
	}
	log.Println("MONGODB is ", MONGODB)

	AWSBucket = os.Getenv("AWSBucket")
	if AWSBucket == "" {
		log.Println("No AWSBucket set, resulting to default")
		AWSBucket = "oddjobz"
	}
	log.Println("AWSBucket is ", AWSBucket)

	Public, err := ioutil.ReadFile("app.rsa.pub")
	if err != nil {
		log.Fatal("Error reading public key")
		return
	}

	Private, err = ioutil.ReadFile("app.rsa")
	if err != nil {
		log.Fatal("Error reading private key")
		return
	}

	RootURL = os.Getenv("RootURL")
	if RootURL == "" {
		RootURL = "http://localhost:8080"
	}

	return
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	REDISADDR, REDISPW, MONGOSERVER, MONGODB, Public, Private, RootURL, AWSBucket := checks()
	session, err := mgo.Dial(MONGOSERVER)
	if err != nil {
		panic(err)
	}
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)

	auth, err := aws.EnvAuth()
	if err != nil {
		//panic(err)
		log.Println("no aws ish")
	}
	s := s3.New(auth, aws.USWest2)
	s3bucket := s.Bucket(AWSBucket)
	rediscli := redis.NewClient(&redis.Options{
		Addr:     REDISADDR,
		Network:  "tcp",
		Password: REDISPW,
	})
	pong, err := rediscli.Ping().Result()
	log.Println(pong, err)
	if err != nil {
		panic(err)

	}
	appC := appContext{
		db:        session.DB(MONGODB),
		verifyKey: []byte(Public),
		signKey:   []byte(Private),
		token:     "AccessToken",
		domain:    RootURL,
		bucket:    s3bucket,
		redis:     rediscli,
	}
	commonHandlers := alice.New(context.ClearHandler, loggingHandler, recoverHandler)
	router := NewRouter()

	router.Post("/api/v0.1/auth", commonHandlers.ThenFunc(appC.authHandler))

	router.Get("/api/v0.1/skills/:slug/reviews", commonHandlers.ThenFunc(appC.reviewsHandler))
	router.Post("/api/v0.1/skills/:slug/reviews", commonHandlers.Append(appC.frontAuthHandler, bodyHandler(ReviewResource{})).ThenFunc(appC.newReviewHandler))

	router.Get("/api/v0.1/skills/:slug", commonHandlers.Append(appC.frontAuthHandler).ThenFunc(appC.skillHandler))
	router.Put("/api/v0.1/skills/:slug", commonHandlers.Append(appC.frontAuthHandler, bodyHandler(SkillResource{})).ThenFunc(appC.updateSkillHandler))
	router.Post("/api/v0.1/skills/:slug", commonHandlers.Append(appC.frontAuthHandler, bodyHandler(SkillResource{})).ThenFunc(appC.updateSkillHandler))

	router.Delete("/api/v0.1/skills/:slug", commonHandlers.ThenFunc(appC.deleteSkillHandler))
	router.Get("/api/v0.1/skills", commonHandlers.Append(appC.frontAuthHandler).ThenFunc(appC.skillsHandler))
	router.Post("/api/v0.1/skills", commonHandlers.Append(appC.frontAuthHandler, bodyHandler(SkillResource{})).ThenFunc(appC.createSkillHandler))

	router.Get("/api/v0.1/user/:username/feeds", commonHandlers.Append(appC.frontAuthHandler).ThenFunc(appC.userFeedsHandler))

	router.Get("/api/v0.1/user/:username/follow", commonHandlers.Append(appC.frontAuthHandler).ThenFunc(appC.followUserHandler))
	router.Get("/api/v0.1/user/:username", commonHandlers.ThenFunc(appC.userHandler))

	router.Get("/api/v0.1/me", commonHandlers.Append(appC.frontAuthHandler).ThenFunc(appC.meHandler))

	router.Get("/api/v0.1/me/feeds", commonHandlers.Append(appC.frontAuthHandler).ThenFunc(appC.userFeedsHandler))

	PORT := os.Getenv("PORT")
	if PORT == "" {
		log.Println("No Global port has been defined, using default")

		PORT = "8080"

	}

	log.Println("serving ")
	log.Fatal(http.ListenAndServe(":"+PORT, router))
}
