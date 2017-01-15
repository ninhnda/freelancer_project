package controllers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/unrolled/render"
	"gitlab.com/personallog/backend/helpers"
	"gitlab.com/personallog/backend/middlewares"
	"gitlab.com/personallog/backend/models"
)

//FeedsSearchModel model post for search feeds
type FeedsSearchModel struct {
	SearchValue string `json:"searchValue"`
}

//FeedsCtrl User logs
type FeedsCtrl struct{}

//List Search list user logs
func (feedsCtrl FeedsCtrl) List(res http.ResponseWriter, req *http.Request) {
	r := render.New(render.Options{})

	vars := mux.Vars(req)
	searchValue := vars["searchValue"]

	feedModel := models.Feed{}
	feed, err := feedModel.Search(searchValue, middlewares.CurrentUser.ID)
	if err != nil {
		r.JSON(res, 500, helpers.GenerateErrorResponse(err.Error(), req.Header))
		return
	}
	r.JSON(res, 200, feed)
}

//Create Feed
func (feedsCtrl FeedsCtrl) Create(res http.ResponseWriter, req *http.Request) {
	r := render.New(render.Options{})

	var feed models.Feed

	// upload file
	req.ParseMultipartForm(0)

	file, handler, err := req.FormFile("file")
	if err != nil {
		feed.Media = ""
	} else {
		defer file.Close()
		var asset models.Asset
		if err := asset.Create(models.AssetTypeImage, handler, file); err != nil {
			r.JSON(res, 422, helpers.GenerateErrorResponse(err.Error(), req.Header))
			return
		}

		if err := asset.Upload(); err != nil {
			r.JSON(res, 500, helpers.GenerateErrorResponse(err.Error(), req.Header))
			return
		}
		feed.Media = asset.Path
		// add tags video/image
		fileExt := filepath.Ext(asset.Path)
		if fileExt == ".jpg" || fileExt == ".png" {
			feed.HashTags = append(feed.HashTags, "image")
		}
		if fileExt == ".mp4" {
			feed.HashTags = append(feed.HashTags, "video")
		}
	}

	// text and another information
	text := req.FormValue("text")
	tags := req.FormValue("tags")

	feed.Text = text
	fmt.Print(tags)
	if len(tags) != 0 {
		feed.HashTags = strings.Split(tags, " ")
	} else {
		feed.HashTags = make([]string, 0)
	}
	feed.CreatedAt = time.Now().Unix()

	creator := models.UserFeed{}
	creator.ID = middlewares.CurrentUser.ID
	creator.EMail = middlewares.CurrentUser.EMail
	creator.Nickname = middlewares.CurrentUser.Nickname
	creator.Picture = middlewares.CurrentUser.Picture

	feed.Creator = creator

	if err := feed.Save(); err != nil {
		r.JSON(res, 500, helpers.GenerateErrorResponse(err.Error(), req.Header))
		return
	}

	r.JSON(res, 200, feed)
	return
}

//Delete delete a Feed
func (feedsCtrl FeedsCtrl) Delete(res http.ResponseWriter, req *http.Request) {
	r := render.New(render.Options{})

	vars := mux.Vars(req)
	feedID := vars["id"]

	var feed models.Feed

	if err := feed.FindByID(feedID); err != nil {
		r.JSON(res, 404, helpers.GenerateErrorResponse(models.ErrRecordNotFound.Error(), req.Header))
		return
	}

	if middlewares.CurrentUser.ID != feed.Creator.ID {
		r.JSON(res, 401, "Not authorization!")
	}

	if err := feed.Delete(feedID); err != nil {
		r.JSON(res, 500, helpers.GenerateErrorResponse(models.ErrRecordNotFound.Error(), req.Header))
		return
	}

	r.Text(res, 204, "")
}
