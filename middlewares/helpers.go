package middlewares

import (
	"net/http"

	"errors"

	"gitlab.com/personallog/backend/helpers"
	"gitlab.com/personallog/backend/models"
)

//ContextKeyType used to remove lint error
type ContextKeyType string

var (
	userKey  ContextKeyType = "user"
	tokenKey ContextKeyType = "token"
)

//CurrentUser current user storage in token
var CurrentUser = models.User{}

//GetTokenFromContext ensures a valid user is received from the context
func GetTokenFromContext(_ http.ResponseWriter, req *http.Request) (string, error) {
	token := req.Context().Value("token").(string)
	if token != "" {
		return token, nil
	}
	return "", errors.New("couldn't find token")
}

//GetUserFromContext ensures a valid user is received from the context
func GetUserFromContext(_ http.ResponseWriter, req *http.Request) (models.User, error) {
	user, ok := req.Context().Value(userKey).(models.User)
	if user.ID == "" || !ok {
		return models.User{}, helpers.ErrUnauthorized
	}
	return user, nil
}
