package main

import (
	"encoding/base64"
	"errors"
	"os"

	"github.com/codegangsta/negroni"
	"github.com/daseinhorn/negroni-json-recovery"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/nicksnyder/go-i18n/i18n"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"gitlab.com/personallog/backend/controllers"
	"gitlab.com/personallog/backend/helpers"
	"gitlab.com/personallog/backend/middlewares"
)

var log *logrus.Logger

func isProdEnv() bool {
	return os.Getenv("APP_ENV") == "prod"
}

func bootstrap() {
	if isProdEnv() {
		err := godotenv.Load("prod.env", "main.env")
		if err != nil {
			panic(err.Error())
		}
	} else {
		err := godotenv.Load("main.env")
		if err != nil {
			panic(err.Error())
		}
	}

	log = helpers.GetLogger()

	// load translations
	i18n.MustLoadTranslationFile("assets/i18n/en-US.all.json")
}

func reportToSentry(err interface{}) {
	var panicErr error
	if e, ok := err.(error); ok {
		panicErr = e
	} else {
		panicErr = errors.New("negroni captured an error but was not able to convert interface to error")
	}
	log.Error(panicErr)
}

func main() {
	bootstrap()

	decodedAuthSecret, decodingErr := base64.URLEncoding.DecodeString(os.Getenv("AUTH0_CLIENT_SECRET"))
	if decodingErr != nil {
		log.Panic(decodingErr)
	}

	// JWT middleware for auth0
	authMiddleware := middlewares.NewAuth(func(token *jwt.Token) (interface{}, error) {
		// if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
		// 	return "", nil
		// }
		return decodedAuthSecret, nil
	}, jwt.SigningMethodHS256)

	baseRouter := mux.NewRouter()

	//controllers v1
	userCtrl := controllers.UsersCtrl{}
	postCtrl := controllers.FeedsCtrl{}

	//user API
	userRouter := mux.NewRouter()

	// userRouter.HandleFunc("/api/v1/users", userCtrl.List).Methods("GET")
	userRouter.HandleFunc("/api/v1/users/profile", userCtrl.Profile).Methods("GET")
	// userRouter.HandleFunc("/api/v1/users", userCtrl.Create).Methods("POST")

	baseRouter.PathPrefix("/api/v1/users").Handler(negroni.New(
		negroni.HandlerFunc(authMiddleware.IsAuthenticated),
		negroni.Wrap(userRouter),
	))

	//post API
	postRouter := mux.NewRouter()
	postRouter.HandleFunc("/api/v1/feeds", postCtrl.Create).Methods("POST")
	postRouter.HandleFunc("/api/v1/feeds", postCtrl.List).Methods("GET")
	postRouter.HandleFunc("/api/v1/feeds/{searchValue}", postCtrl.List).Methods("GET")
	postRouter.HandleFunc("/api/v1/feeds/{id}", postCtrl.Delete).Methods("DELETE")

	baseRouter.PathPrefix("/api/v1/feeds").Handler(negroni.New(
		negroni.HandlerFunc(authMiddleware.IsAuthenticated),
		negroni.Wrap(postRouter),
	))

	//negroni routing setup
	n := negroni.New()

	sentryRecovery := negroni.NewRecovery()
	sentryRecovery.ErrorHandlerFunc = reportToSentry
	n.Use(sentryRecovery)

	//disable logging if not in dev mode
	if !isProdEnv() || true {
		n.Use(recovery.JSONRecovery(true))
		n.Use(negroni.NewLogger())
	}

	// configure cors settings
	options := cors.Options{
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowedHeaders:   []string{"*"},
	}

	if isProdEnv() {
		options.AllowedOrigins = []string{
		//prod configs
		// "https://nowatwork.ch",
		// "https://control.nowatwork.ch",
		// "https://naw-control.ch",
		}
		options.Debug = false
	} else {
		options.AllowedOrigins = []string{"*"}
		options.Debug = false
	}
	n.Use(cors.New(options))

	n.UseHandler(baseRouter)

	// Get PORT from env
	port := os.Getenv("PORT")
	if port == "" {
		port = "3008"
	}

	n.Run(":" + port)
}
