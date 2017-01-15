package controllers

import (
	"encoding/json"
	"net/http"

	"gitlab.com/personallog/backend/helpers"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/unrolled/render"
	"gitlab.com/personallog/backend/models"
	"gitlab.com/personallog/backend/middlewares"
)

//UsersCtrl is the controller for /users
type UsersCtrl struct{}

func (usersCtrl UsersCtrl) getLogger() *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		"controller": "UsersCtrl",
	})
}

//Profile get profile's current user
func (usersCtrl UsersCtrl) Profile(res http.ResponseWriter, req *http.Request) {
	r := render.New(render.Options{})
	r.JSON(res, 200, middlewares.CurrentUser)
}

//List all users
func (usersCtrl UsersCtrl) List(res http.ResponseWriter, req *http.Request) {
	r := render.New(render.Options{})

	vars := mux.Vars(req)
	id := vars["id"]

	var user models.User

	if id != "" {
		if err := user.FindByID(id); err != nil {
			jsonErr := helpers.GenerateJSONError(err, req.Header)
			r.JSON(res, jsonErr.StatusCode, jsonErr)
			return
		}

		r.JSON(res, 200, user)
		return
	}

	users, err := user.FindAll()
	if err != nil {
		r.JSON(res, 500, helpers.GenerateErrorResponse(err.Error(), req.Header))
		return
	}

	r.JSON(res, 200, users)
	return
}

type userCreateRequestModel struct {
	EMail    string `json:"email"`
	Password string `json:"password"`
}

//Create user
func (usersCtrl UsersCtrl) Create(res http.ResponseWriter, req *http.Request) {
	r := render.New(render.Options{})

	decoder := json.NewDecoder(req.Body)
	var requestData userCreateRequestModel
	if err := decoder.Decode(&requestData); err != nil {
		jsonErr := helpers.GenerateJSONError(helpers.ErrBadRequest, req.Header)
		r.JSON(res, jsonErr.StatusCode, jsonErr)
		return
	}

	var user models.User
	if err := user.Create(requestData.EMail, requestData.Password); err != nil {
		jsonErr := helpers.GenerateJSONError(err, req.Header)
		r.JSON(res, jsonErr.StatusCode, jsonErr)
		return
	}

	r.JSON(res, 200, user)
	return
}

//Update user
func (usersCtrl UsersCtrl) Update(res http.ResponseWriter, req *http.Request) {
	r := render.New(render.Options{})

	vars := mux.Vars(req)
	userID := vars["id"]

	if userID == "" {
		jsonErr := helpers.GenerateJSONError(helpers.ErrBadRequest, req.Header)
		r.JSON(res, jsonErr.StatusCode, jsonErr)
		return
	}

	var requestData models.UserUpdateRequestModel

	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&requestData); err != nil {
		jsonErr := helpers.GenerateJSONError(helpers.ErrBadRequest, req.Header)
		r.JSON(res, jsonErr.StatusCode, jsonErr)
		return
	}

	var user models.User
	if err := user.FindByID(userID); err != nil {
		jsonErr := helpers.GenerateJSONError(err, req.Header)
		r.JSON(res, jsonErr.StatusCode, jsonErr)
		return
	}

	if err := user.Update(requestData); err != nil {
		jsonErr := helpers.GenerateJSONError(err, req.Header)
		r.JSON(res, jsonErr.StatusCode, jsonErr)
		return
	}

	r.JSON(res, 200, user)
}

//Delete user
func (usersCtrl UsersCtrl) Delete(res http.ResponseWriter, req *http.Request) {
	r := render.New(render.Options{})

	vars := mux.Vars(req)
	userID := vars["id"]

	if userID == "" {
		jsonErr := helpers.GenerateJSONError(helpers.ErrBadRequest, req.Header)
		r.JSON(res, jsonErr.StatusCode, jsonErr)
		return
	}

	var user models.User
	if err := user.DeleteByID(userID); err != nil {
		jsonErr := helpers.GenerateJSONError(err, req.Header)
		r.JSON(res, jsonErr.StatusCode, jsonErr)
		return
	}

	r.Text(res, 204, "")
}
