package models

import (
	"os"

	"fmt"

	"strconv"

	"github.com/algolia/algoliasearch-client-go/algoliasearch"
)

// Database unit
type Database struct {
	indexName string
}

//Save insert or update record
func (database *Database) Save(data map[string]interface{}) error {

	index := database.GetIndex()
	_, err := index.UpdateObject(data)

	return err
}

//Query search record from database
func (database *Database) Query(keyword string, searchType int, userID string) (algoliasearch.QueryRes, error) {

	fmt.Print("key word : " + keyword)
	fmt.Print("search type : " + strconv.Itoa(searchType))
	fmt.Print("userID : " + userID)

	index := database.GetIndex()
	searchValue := ""

	// parse filters to string
	filtersStr := "creator.id:\"" + userID + "\""

	if searchType == 1 { // search by date
		filtersStr += " AND createdAt=" + keyword
	} else if searchType == 2 {
		filtersStr += " AND _tags:\"" + keyword + "\""
	} else if searchType == 3 { // search text normal
		searchValue = keyword
	}
	params := algoliasearch.Map{
		"hitsPerPage": 50,
		"filters":     filtersStr,
		"facets":      "*",
	}

	res, err := index.Search(searchValue, params)
	return res, err
}

//Delete delete record
func (database *Database) Delete(id string) error {
	index := database.GetIndex()
	_, err := index.DeleteObject(id)
	return err
}

//GetByID get an object by objectId
func (database *Database) GetByID(id string) (algoliasearch.Object, error) {
	index := database.GetIndex()
	object, err := index.GetObject(id, nil)

	return object, err
}

//GetIndex get index by name
func (database Database) GetIndex() algoliasearch.Index {
	clientID := os.Getenv("ALGOLIA_CLIENT_ID")
	clientSecret := os.Getenv("ALGOLIA_CLIENT_SECRET")

	client := algoliasearch.NewClient(clientID, clientSecret)
	index := client.InitIndex(database.indexName)

	return index
}
