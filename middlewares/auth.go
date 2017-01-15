package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"errors"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/unrolled/render"
	"gitlab.com/personallog/backend/helpers"
	"gitlab.com/personallog/backend/models"
)

// current user
// var currentUser = models.User{}

// export

//Auth used to secure the API
type Auth struct {
	KeyFunc       jwt.Keyfunc
	SigningMethod jwt.SigningMethod
}

//NewAuth generates a new instance
func NewAuth(keyFunc jwt.Keyfunc, signingMethod jwt.SigningMethod) Auth {
	return Auth{
		KeyFunc:       keyFunc,
		SigningMethod: signingMethod,
	}
}

func (auth Auth) getTokenFromHeader(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", nil // No error, just no token
	}
	authHeaderParts := strings.Split(authHeader, " ")
	if len(authHeaderParts) != 2 || strings.ToLower(authHeaderParts[0]) != "bearer" {
		return "", errors.New("Authorization header format must be Bearer {token}")
	}
	return authHeaderParts[1], nil
}

//getAuth0UserByToken uses the Auth0 API to fetch the user information
//it is both faster and more secure.
func (auth Auth) getAuth0UserByToken(token *jwt.Token) (models.User, error) {
	if !token.Valid || token.Claims.Valid() != nil {
		return models.User{}, helpers.ErrInvalidToken
	}

	var user models.User
	if err := user.FindByJWTToken(token); err != nil {
		return models.User{}, helpers.ErrInvalidToken
	}

	return user, nil
}

//GetUserID Get current user id
func (auth *Auth) GetUserID(req *http.Request) (string, error) {
	tokenStr, err := auth.getTokenFromHeader(req)
	if err != nil {
		return "", err
	}

	token, err := jwt.Parse(tokenStr, auth.KeyFunc)
	if err != nil {
		return "", err
	}

	user, err := auth.getAuth0UserByToken(token)
	if err != nil {
		return "", err
	}

	return user.ID, nil
}

//IsAuthenticated passes user to the request context
func (auth *Auth) IsAuthenticated(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
	tokenStr, err := auth.getTokenFromHeader(req)
	if tokenStr == "" || err != nil {
		fmt.Print(err)
		r := render.New(render.Options{})
		jsonErr := helpers.GenerateJSONError(helpers.ErrInvalidToken, req.Header)
		r.JSON(res, jsonErr.StatusCode, jsonErr)
		return
	}

	token, err := jwt.Parse(tokenStr, auth.KeyFunc)
	if err != nil || !token.Valid {
		fmt.Print(err)
		r := render.New(render.Options{})
		jsonErr := helpers.GenerateJSONError(helpers.ErrInvalidToken, req.Header)
		r.JSON(res, jsonErr.StatusCode, jsonErr)
		return
	}
	user, err := auth.getAuth0UserByToken(token)

	CurrentUser = user

	ctx := context.WithValue(req.Context(), tokenKey, token)
	ctx = context.WithValue(ctx, userKey, user)

	next(res, req.WithContext(ctx))
	return
}

//IsConsultant can be used AFTER IsAuthenticatedMiddleware to check if the user is a consultant
func (auth *Auth) IsConsultant(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
	user, err := GetUserFromContext(res, req)
	if err != nil || !user.IsSupervisor() || !user.EMailVerified {
		r := render.New(render.Options{})
		jsonErr := helpers.GenerateJSONError(helpers.ErrUnauthorized, req.Header)
		r.JSON(res, jsonErr.StatusCode, jsonErr)
		return
	}
	next(res, req)
	return
}

//IsSupervisor can be used AFTER IsAuthenticatedMiddleware to check if the user is a supervisor
func (auth *Auth) IsSupervisor(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
	user, err := GetUserFromContext(res, req)
	if err != nil || !user.IsSupervisor() || !user.EMailVerified {
		r := render.New(render.Options{})
		jsonErr := helpers.GenerateJSONError(helpers.ErrUnauthorized, req.Header)
		r.JSON(res, jsonErr.StatusCode, jsonErr)
		return
	}
	next(res, req)
	return
}

//IsAdmin can be used AFTER IsAuthenticatedMiddleware to check if the user is admin
func (auth *Auth) IsAdmin(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
	user, err := GetUserFromContext(res, req)
	if err != nil || !user.IsAdmin() || !user.EMailVerified {
		r := render.New(render.Options{})
		jsonErr := helpers.GenerateJSONError(helpers.ErrUnauthorized, req.Header)
		r.JSON(res, jsonErr.StatusCode, jsonErr)
		return
	}
	next(res, req)
	return
}
