package models

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"time"

	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
	"gitlab.com/personallog/backend/helpers"
	"gopkg.in/asaskevich/govalidator.v4"
	"gopkg.in/mgo.v2/bson"
)

const (
	//UserModelRoleParticipant for general low access roles
	UserModelRoleParticipant = 1
	//UserModelRoleConsultant for general genearl consultant access
	UserModelRoleConsultant = 2
)

var (
	groupNameAdmin      = "admin"
	groupNameSuperAdmin = "headbits"
	groupNameProduction = "production"
)

type userAuthorization struct {
	Groups       []string `json:"groups"`
	IsAdmin      bool     `json:"isAdmin"`
	IsSuperAdmin bool     `json:"isSuperAdmin"`
}

type userAuthorizationPermissionsMap map[string]userAuthorizationPermission

type userAuthorizationPermission struct {
	Role   string   `json:"role"`
	Groups []string `json:"groups"`
}

//User model
type User struct {
	ID string `json:"id"`

	Nickname string `json:"nickname"`
	Picture  string `json:"picture"`

	EMail         string `json:"email"`
	EMailVerified bool   `json:"emailVerified"`

	LastIP      string    `json:"lastIP"`
	LastLogin   time.Time `json:"lastLogin"`
	LoginsCount int       `json:"loginsCount"`

	Blocked bool `json:"blocked"`

	Permissions userAuthorizationPermission `json:"permissions"`

	UserMetadata userAuth0UserMetadata `json:"-"`
	AppMetadata  userAuth0AppMetadata  `json:"-"`
}

// UserTokenInfor User model parse from token
type UserTokenInfor struct {
	Name     string `json:"name"`
	Nickname string `json:"nickname"`
	Picture  string `json:"picture"`
	UpdateAt string `json:"updated_at"`
	Email    string `json:email`
	Sub      string `json:sub`
}

//getLogger
func (user User) getLogger() *logrus.Entry {
	return helpers.GetLogger().WithFields(logrus.Fields{
		"model": "User",
	})
}

//Validate the given question
func (user *User) Validate() error {
	_, err := govalidator.ValidateStruct(user)
	if err != nil {
		return err
	}

	return nil
}

//convertFromAuth0 users
func (user *User) convertFromAuth0(auth0User userAuth0UserResponseModel) error {
	user.ID = strings.Replace(auth0User.UserID, "auth0|", "", -1)

	user.Nickname = auth0User.Nickname

	user.Picture = auth0User.Picture

	user.EMail = auth0User.Email
	user.EMailVerified = auth0User.EmailVerified

	user.LastIP = auth0User.LastIP
	user.LastLogin = auth0User.LastLogin
	user.LoginsCount = auth0User.LoginsCount

	user.Blocked = auth0User.Blocked

	user.UserMetadata = auth0User.UserMetadata
	user.AppMetadata = auth0User.AppMetadata

	// user.GetPermissions()

	return nil
}

//IsAuthenticated checks if the user has access to the tenant
func (user *User) IsAuthenticated() bool {
	return !user.IsRole(os.Getenv("AUTH_DEFAULT_UNAUTHORIZED"))
}

//GetPermissions returns permissions for the current tenant
func (user *User) GetPermissions() error {
	if len(user.AppMetadata.Permissions) == 0 {
		user.AppMetadata.Permissions = make(userAuthorizationPermissionsMap)
	}
	permissions, ok := user.AppMetadata.Permissions[helpers.GetAuthTenant()]
	if !ok {
		permissions = userAuthorizationPermission{
			Role:   os.Getenv("AUTH_DEFAULT_UNAUTHORIZED"),
			Groups: []string{},
		}
	}
	user.AppMetadata.Permissions[helpers.GetAuthTenant()] = permissions
	user.Permissions = permissions
	return nil
}

//GetRole returns the users role
func (user User) GetRole() string {
	return user.AppMetadata.Permissions[helpers.GetAuthTenant()].Role
}

//IsRole returns true on roles match
func (user User) IsRole(roleName string) bool {
	return user.GetRole() == roleName
}

//IsConsultant returns true if user is either a consultant, supervisor or admin
func (user User) IsConsultant() bool {
	return user.IsRole(os.Getenv("AUTH_ROLE_CONSULTANT")) || user.IsRole(os.Getenv("AUTH_ROLE_SUPERVISOR")) || user.IsRole(os.Getenv("AUTH_ROLE_ADMIN"))
}

//IsSupervisor returns true if user is a supervisor or admin
func (user User) IsSupervisor() bool {
	return user.IsRole(os.Getenv("AUTH_ROLE_SUPERVISOR")) || user.IsRole(os.Getenv("AUTH_ROLE_ADMIN"))
}

//IsAdmin returns true if user is a admin
func (user *User) IsAdmin() bool {
	return user.IsRole(os.Getenv("AUTH_ROLE_ADMIN"))
}

//UpdateRole sets the role for the current tenant
func (user *User) UpdateRole(newRole string) error {
	if newRole == "" {
		newRole = os.Getenv("AUTH_DEFAULT_ROLE")
	}

	if len(user.AppMetadata.Permissions) == 0 {
		user.AppMetadata.Permissions = make(userAuthorizationPermissionsMap)
	}

	permissions, ok := user.AppMetadata.Permissions[helpers.GetAuthTenant()]
	if !ok {
		permissions = userAuthorizationPermission{
			Role:   newRole,
			Groups: []string{},
		}
	}
	permissions.Role = newRole
	user.AppMetadata.Permissions[helpers.GetAuthTenant()] = permissions
	return nil
}

//FindAll users
func (user *User) FindAll() ([]User, error) {
	url := os.Getenv("AUTH0_DOMAIN") + "/api/v2/users"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	authToken, err := getAuth0Token(false)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", authToken)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if err := checkAuth0Response(url, res); err != nil {
		return nil, err
	}

	var auth0Users []userAuth0UserResponseModel

	if err := json.NewDecoder(res.Body).Decode(&auth0Users); err != nil {
		return nil, err
	}

	var users []User
	for _, auth0User := range auth0Users {
		var user User
		user.convertFromAuth0(auth0User)
		users = append(users, user)
	}

	return users, nil
}

//FindByJWTToken get user by JWT token
func (user *User) FindByJWTToken(token *jwt.Token) error {
	if !token.Valid {
		return helpers.ErrInvalidToken
	}

	tokenStr := token.Raw
	claims := strings.Split(tokenStr, ".")
	userInforStr, err := jwt.DecodeSegment(claims[1])

	if err != nil {
		return err
	}

	var userInfor UserTokenInfor

	err = json.Unmarshal(userInforStr, &userInfor)
	if err != nil {
		return err
	}

	user.ID = strings.Replace(userInfor.Sub, "auth0|", "", -1)
	user.Nickname = userInfor.Nickname
	user.Picture = userInfor.Picture
	user.EMail = userInfor.Email

	return nil
}

//FindByID user
func (user *User) FindByID(id string) error {
	url := os.Getenv("AUTH0_DOMAIN") + "/api/v2/users/" + url.QueryEscape(id)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	authToken, err := getAuth0Token(false)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", authToken)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if err := checkAuth0Response(url, res); err != nil {
		return err
	}

	var auth0User userAuth0UserResponseModel

	if err := json.NewDecoder(res.Body).Decode(&auth0User); err != nil {
		return err
	}

	user.convertFromAuth0(auth0User)

	return nil
}

func (user User) generateRandomPassword() string {
	b := make([]byte, 6)
	rand.Read(b)
	return fmt.Sprintf("%x", b) + "Aa12$%"
}

//Create a user at auth0
func (user *User) Create(eMail string, password string) error {
	url := os.Getenv("AUTH0_DOMAIN") + "/api/v2/users"

	payloadData := userAuth0UserCreateRequestModel{
		Connection:    "Username-Password-Authentication",
		EMail:         eMail,
		EMailVerified: true,
	}

	//generate password if empty
	if password == "" {
		password = user.generateRandomPassword()
	}
	payloadData.Password = password

	//generate permissions data
	user.GetPermissions()
	payloadData.AppMetadata = user.AppMetadata

	payload, err := json.Marshal(payloadData)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	authToken, err := getAuth0Token(false)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", authToken)
	req.Header.Add("content-type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	fmt.Print(res.StatusCode)

	if res.StatusCode == http.StatusCreated {
		var auth0User userAuth0UserResponseModel

		if err := json.NewDecoder(res.Body).Decode(&auth0User); err != nil {
			fmt.Print("lalalalal")
			return err
		}
		fmt.Print("==================+++++++++++wefwff++++++++++")
		user.convertFromAuth0(auth0User)
		return nil
	}

	var auth0Err helpers.JSONError
	if err := json.NewDecoder(res.Body).Decode(&auth0Err); err != nil {
		return err
	}

	user.getLogger().WithFields(logrus.Fields{
		"URL": url,
	}).Error(auth0Err)

	return errors.New(auth0Err.ErrorCode)
}

//UserUpdateRequestModel is used as abstraction between the client facting API and auth0
type UserUpdateRequestModel struct {
	CompanyObjectID bson.ObjectId `json:"companyId,omitempty"`
	Blocked         bool          `json:"blocked,omitempty"`
	Email           string        `json:"email,omitempty"`
	Password        string        `json:"password,omitempty"`
}

//Update user
func (user *User) Update(updateRequest UserUpdateRequestModel) error {
	if user.ID == "" {
		return helpers.ErrInvalidObjectID
	}

	reqData := userAuth0UserUpdateRequestModel{}

	reqData.Blocked = user.Blocked

	//check if EMail has changed
	if user.EMail != updateRequest.Email {
		reqData.Email = updateRequest.Email
	}

	if updateRequest.Password != "" {
		reqData.Password = updateRequest.Password
	}

	user.UpdateRole(user.GetRole())
	reqData.AppMetadata = user.AppMetadata

	url := os.Getenv("AUTH0_DOMAIN") + "/api/v2/users/" + url.QueryEscape(user.ID)

	payload, err := json.Marshal(reqData)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	authToken, err := getAuth0Token(false)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", authToken)
	req.Header.Add("content-type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if err := checkAuth0Response(url, res); err != nil {
		return err
	}

	var auth0User userAuth0UserResponseModel
	if err := json.NewDecoder(res.Body).Decode(&auth0User); err != nil {
		return err
	}

	user.convertFromAuth0(auth0User)
	return nil
}

//Delete a user
func (user *User) Delete() error {
	return user.DeleteByID(user.ID)
}

//DeleteByID a user by id
func (user User) DeleteByID(userID string) error {
	url := os.Getenv("AUTH0_DOMAIN") + "/api/v2/users/" + url.QueryEscape(userID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	authToken, err := getAuth0Token(false)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", authToken)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if err := checkAuth0Response(url, res); err != nil {
		return err
	}

	return nil
}
