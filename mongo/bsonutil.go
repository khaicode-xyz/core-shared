package mongo

import (
	"regexp"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ToObjectID parses a hex string. Returns NilObjectID when the input
// is malformed (matches Foxel.Mongo.BsonUtils.ToObjectId).
func ToObjectID(value string) bson.ObjectID {
	if value == "" {
		return bson.NilObjectID
	}
	oid, err := bson.ObjectIDFromHex(value)
	if err != nil {
		return bson.NilObjectID
	}
	return oid
}

// IsObjectIDEmpty reports whether the hex string parses to NilObjectID
// (or fails to parse at all).
func IsObjectIDEmpty(value string) bool {
	return ToObjectID(value) == bson.NilObjectID
}

// StartsWith builds a `{field: /^value/}` regex filter.
// Mirrors Foxel.Mongo.BsonUtils.StartWith.
func StartsWith(field, value string) bson.M {
	return bson.M{field: bson.M{"$regex": bson.Regex{Pattern: "^" + regexp.QuoteMeta(value)}}}
}

// EndsWith builds a `{field: /value$/}` regex filter.
func EndsWith(field, value string) bson.M {
	return bson.M{field: bson.M{"$regex": bson.Regex{Pattern: regexp.QuoteMeta(value) + "$"}}}
}

// Contains builds a `{field: /value/}` regex filter.
func Contains(field, value string) bson.M {
	return bson.M{field: bson.M{"$regex": bson.Regex{Pattern: regexp.QuoteMeta(value)}}}
}
