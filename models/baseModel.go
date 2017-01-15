package models

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

//BaseModel is used as mgo.Model replacement
type BaseModel struct {
	ID        bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
	CreatedAt int64         `json:"createdAt,omitempty" bson:"createdAt,omitempty"`
	UpdatedAt int64         `json:"updatedAt,omitempty" bson:"updatedAt,omitempty"`
}

//SetID generates a new ObjectId and sets the CreatedAt
//if the ID is empty or invalid
func (baseModel *BaseModel) SetID() {
	if baseModel.ID.Hex() == "" || !baseModel.ID.Valid() {
		baseModel.ID = bson.NewObjectId()
		baseModel.CreatedAt = time.Now().Unix()
	}
}
