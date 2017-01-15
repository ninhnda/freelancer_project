package models

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/mgo.v2/bson"

	"github.com/sirupsen/logrus"
	"gitlab.com/personallog/backend/helpers"
)

var auth0ManagementToken = ""

type userAuth0TokenResponseModel struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type userAuth0TokenInfoRequestModel struct {
	Token string `json:"id_token"`
}

type userAuth0UserCreateRequestModel struct {
	Connection    string                `json:"connection"`
	EMail         string                `json:"email"`
	EMailVerified bool                  `json:"email_verified"`
	Password      string                `json:"password"`
	UserMetadata  userAuth0UserMetadata `json:"user_metadata,omitempty"`
	AppMetadata   userAuth0AppMetadata  `json:"app_metadata,omitempty"`
}

type userAuth0UserUpdateRequestModel struct {
	Blocked           bool                  `json:"blocked"`
	EmailVerified     bool                  `json:"email_verified,omitempty"`
	Email             string                `json:"email,omitempty"`
	VerifyEmail       bool                  `json:"verify_email,omitempty"`
	PhoneNumber       string                `json:"phone_number,omitempty"`
	PhoneVerified     bool                  `json:"phone_verified,omitempty"`
	VerifyPhoneNumber bool                  `json:"verify_phone_number,omitempty"`
	Password          string                `json:"password,omitempty"`
	VerifyPassword    bool                  `json:"verify_password,omitempty"`
	UserMetadata      userAuth0UserMetadata `json:"user_metadata,omitempty"`
	AppMetadata       userAuth0AppMetadata  `json:"app_metadata,omitempty"`
	Connection        string                `json:"connection,omitempty"`
	Username          string                `json:"username,omitempty"`
	ClientID          string                `json:"client_id,omitempty"`
}

type userAuth0AppMetadata struct {
	Permissions userAuthorizationPermissionsMap `json:"permissions"`
}

type userAuth0UserMetadata struct {
	CompanyObjectID bson.ObjectId `json:"companyObjectId,omitempty"`
	CompanyName     string        `json:"companyName,omitempty"`
}

type userAuth0Identity struct {
	UserID     string `json:"user_id"`
	Provider   string `json:"provider"`
	Connection string `json:"connection"`
	IsSocial   bool   `json:"isSocial"`
}

type userAuth0UserResponseModel struct {
	Name     string `json:"name"`
	Picture  string `json:"picture"`
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname"`

	UserMetadata userAuth0UserMetadata `json:"user_metadata"`
	AppMetadata  userAuth0AppMetadata  `json:"app_metadata"`

	EmailVerified bool   `json:"email_verified"`
	Email         string `json:"email"`

	Blocked bool `json:"blocked"`

	Identities []userAuth0Identity `json:"identities"`

	LastIP      string    `json:"last_ip"`
	LastLogin   time.Time `json:"last_login"`
	LoginsCount int       `json:"logins_count"`

	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
}

//getAuth0Token returns JWT token for admin access
func getAuth0Token(forceRefresh bool) (string, error) {
	if auth0ManagementToken != "" && !forceRefresh {
		return auth0ManagementToken, nil
	}

	url := os.Getenv("AUTH0_DOMAIN") + "/oauth/token"
	payload := strings.NewReader("{\"client_id\":\"8pKXFFLnvgeYAU0TzoCi6hok7WfCwYb9\",\"client_secret\":\"WzDVcA2WclTyvE6fKd_kDQhjL-kNxdYCyaCs8fGzeO8PqrkkbPj8UdrjfOhf_UjG\",\"audience\":\"https://lehoai.auth0.com/api/v2/\",\"grant_type\":\"client_credentials\"}")
	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return "", err
	}

	req.Header.Add("content-type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	var token userAuth0TokenResponseModel
	if err := json.NewDecoder(res.Body).Decode(&token); err != nil {
		return "", err
	}

	parsedToken := token.TokenType + " " + token.AccessToken
	return parsedToken, nil
}

//checks response for error
func checkAuth0Response(url string, res *http.Response) error {
	statusCode := res.StatusCode

	if statusCode == http.StatusOK || statusCode == http.StatusCreated || statusCode == http.StatusNoContent {
		return nil
	}

	if statusCode == http.StatusNotFound {
		return helpers.ErrRecordNotFound
	}
	if statusCode == http.StatusUnauthorized {
		getLogger().WithFields(logrus.Fields{
			"URL": url,
		}).Error(helpers.ErrUnauthorized)
		return helpers.ErrUnauthorized
	}
	if statusCode == http.StatusForbidden {
		return helpers.ErrForbidden
	}

	body, _ := ioutil.ReadAll(res.Body)
	getLogger().WithFields(logrus.Fields{
		"URL": url,
	}).Error(string(body))
	return helpers.ErrRequestTimeOut
}
