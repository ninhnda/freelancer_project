package models

import (
	// "bytes"
	// "encoding/json"
	// "errors"
	// "io/ioutil"
	// "math/rand"
	// "net/http"
	// "net/url"
	// "strings"

	"strings"
	"time"

	"strconv"

	"github.com/algolia/algoliasearch-client-go/algoliasearch"
	"github.com/sirupsen/logrus"
	"gitlab.com/personallog/backend/helpers"
	"gopkg.in/asaskevich/govalidator.v4"
	"gopkg.in/mgo.v2/bson"
)

//CollectionFeedProperty algolia table name
var CollectionFeedProperty = "Feeds"

//UserFeed model for user in feed
type UserFeed struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	Picture  string `json:"picture"`
	EMail    string `json:"email"`
}

//Feed model
type Feed struct {
	BaseModel `bson:",inline"`

	Media    string   `json:"media"`
	Text     string   `json:"text" valid:"required"`
	Creator  UserFeed `json:creator`
	HashTags []string `json:hashTags`
}

func (feed Feed) getLogger() *logrus.Entry {
	return helpers.GetLogger().WithFields(logrus.Fields{
		"model":      "SurveyModule",
		"collection": CollectionFeedProperty,
	})
}

//Save or updates the given object
func (feed *Feed) Save() error {
	_, err := govalidator.ValidateStruct(feed)
	if err != nil {
		return err
	}

	feed.SetID()
	feed.UpdatedAt = time.Now().Unix()

	object := algoliasearch.Object{
		"objectID": feed.ID.Hex(),
		"text":     feed.Text,
		"media":    feed.Media,
		"_tags":    feed.HashTags,
		"creator":  feed.Creator,
		"createAt": feed.CreatedAt,
		"updateAt": feed.UpdatedAt,
	}
	database := Database{CollectionFeedProperty}
	err = database.Save(object)
	return err
}

//Search objects
func (feed *Feed) Search(keyword string, userID string) ([]Feed, error) {
	feeds := make([]Feed, 0)
	database := Database{CollectionFeedProperty}
	searchType := 1

	// convert keyword to query
	if strings.Index(keyword, "#") == 0 { // search by hashtag
		keyword = strings.Replace(keyword, "#", "", -1)
		searchType = 2
	} else if strings.Index(keyword, "date:") == 0 {
		keyword = strings.Replace(keyword, "date:", "", -1)
		searchType = 1
	} else {
		searchType = 3
	}

	res, err := database.Query(keyword, searchType, userID)
	if err != nil {
		return nil, err
	}

	for _, element := range res.Hits {
		tmp := Feed{}
		tmp.ID = bson.ObjectIdHex(element["objectID"].(string))
		tmp.Text = element["text"].(string)
		tmp.Media = element["media"].(string)
		// tmp.Creator = element["creator"].(string)

		tmpUser := element["creator"].(map[string]interface{})
		tmpCreator := UserFeed{}

		tmpCreator.ID = tmpUser["id"].(string)
		tmpCreator.Nickname = tmpUser["nickname"].(string)
		tmpCreator.EMail = tmpUser["email"].(string)

		tmp.Creator = tmpCreator

		createAt := strconv.FormatFloat(element["createAt"].(float64), 'f', 6, 64)
		updateAt := strconv.FormatFloat(element["updateAt"].(float64), 'f', 6, 64)

		tmp.CreatedAt, _ = strconv.ParseInt(strings.Split(createAt, ".")[0], 10, 64)
		tmp.UpdatedAt, _ = strconv.ParseInt(strings.Split(updateAt, ".")[0], 10, 64)

		tmp.HashTags = make([]string, 0)
		if element["_tags"] != nil {
			for _, tag := range element["_tags"].([]interface{}) {
				tmp.HashTags = append(tmp.HashTags, tag.(string))
			}
		}

		feeds = append(feeds, tmp)
	}

	return feeds, nil
}

//FindByID Find a post by ID
func (feed *Feed) FindByID(id string) error {
	database := Database{CollectionFeedProperty}

	element, err := database.GetByID(id)

	if err != nil {
		feed = nil
		return err
	}

	feed.ID = bson.ObjectIdHex(element["objectID"].(string))
	feed.Text = element["text"].(string)
	feed.Media = element["media"].(string)
	creator := element["creator"].(map[string]interface{})
	feed.Creator.ID = creator["id"].(string)

	createAt := strconv.FormatFloat(element["createAt"].(float64), 'f', 6, 64)
	updateAt := strconv.FormatFloat(element["updateAt"].(float64), 'f', 6, 64)

	feed.CreatedAt, _ = strconv.ParseInt(strings.Split(createAt, ".")[0], 10, 64)
	feed.UpdatedAt, _ = strconv.ParseInt(strings.Split(updateAt, ".")[0], 10, 64)

	feed.HashTags = make([]string, 0)
	if element["_tags"] != nil {
		for _, tag := range element["_tags"].([]interface{}) {
			feed.HashTags = append(feed.HashTags, tag.(string))
		}
	}

	return nil
}

//Delete delete post
func (feed *Feed) Delete(id string) error {
	database := Database{CollectionFeedProperty}
	err := database.Delete(id)
	return err
}
