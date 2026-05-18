package mongo

// Entity is the shared MongoDB document base. Mirrors
// Foxel.Mongo.Entities.MongoEntity (without the multi-tenant variant).
//
// Embed Entity in concrete models:
//
//	type Agent struct {
//	    mongo.Entity `bson:",inline"`
//	    Platform     string `bson:"platform" json:"platform"`
//	}
//
// `_id` is exposed as a plain string and round-trips through the
// driver as a BSON ObjectID via StringObjectID.
type Entity struct {
	ID             StringObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Code           string         `bson:"code" json:"code"`
	Name           string         `bson:"name,omitempty" json:"name,omitempty"`
	Description    string         `bson:"description,omitempty" json:"description,omitempty"`
	CreatedBy      string         `bson:"created_by,omitempty" json:"created_by,omitempty"`
	CreatedDate    DateTime       `bson:"created_date" json:"created_date"`
	LastModifiedBy string         `bson:"last_modified_by,omitempty" json:"last_modified_by,omitempty"`
	LastModified   DateTime       `bson:"last_modified" json:"last_modified"`
	Status         string         `bson:"status" json:"status"`
}

// GetCode returns the entity's business code (used by Repository.ReplaceByCode).
func (e Entity) GetCode() string { return e.Code }

// GetID returns the entity's `_id` as a string.
func (e Entity) GetID() string { return string(e.ID) }

// Document is the contract a model must satisfy to be used with Repository[T].
//
// Embed Entity to get a free implementation.
type Document interface {
	GetID() string
	GetCode() string
}
