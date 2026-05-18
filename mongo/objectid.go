package mongo

import (
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// StringObjectID is a string that round-trips as a BSON ObjectID when
// encoded, but accepts either a BSON ObjectID or a BSON string when
// decoded. Mirrors Foxel.Mongo.Serializations.Serializers.StringObjectIdSerializer.
//
// Use this for `_id` fields exposed as plain strings in service models.
type StringObjectID string

// String returns the underlying string.
func (s StringObjectID) String() string { return string(s) }

// IsZero reports whether the value is empty or the zero ObjectID.
func (s StringObjectID) IsZero() bool {
	if s == "" {
		return true
	}
	oid, err := bson.ObjectIDFromHex(string(s))
	if err != nil {
		return false
	}
	return oid == bson.NilObjectID
}

// ObjectID parses the string and returns the ObjectID, or NilObjectID
// when the string is empty or malformed.
func (s StringObjectID) ObjectID() bson.ObjectID {
	if s == "" {
		return bson.NilObjectID
	}
	oid, err := bson.ObjectIDFromHex(string(s))
	if err != nil {
		return bson.NilObjectID
	}
	return oid
}

// MarshalBSONValue implements bson.ValueMarshaler. Empty values are
// emitted as null so MongoDB can auto-generate `_id` on insert.
func (s StringObjectID) MarshalBSONValue() (byte, []byte, error) {
	if s == "" {
		return byte(bson.TypeNull), nil, nil
	}
	oid, err := bson.ObjectIDFromHex(string(s))
	if err != nil {
		t, data, err := bson.MarshalValue(string(s))
		return byte(t), data, err
	}
	t, data, err := bson.MarshalValue(oid)
	return byte(t), data, err
}

// UnmarshalBSONValue implements bson.ValueUnmarshaler. Accepts BSON
// ObjectID, string, or null.
func (s *StringObjectID) UnmarshalBSONValue(typ byte, data []byte) error {
	bt := bson.Type(typ)
	switch bt {
	case bson.TypeNull, bson.TypeUndefined:
		*s = ""
		return nil
	case bson.TypeObjectID:
		var oid bson.ObjectID
		if err := bson.UnmarshalValue(bt, data, &oid); err != nil {
			return err
		}
		*s = StringObjectID(oid.Hex())
		return nil
	case bson.TypeString:
		var v string
		if err := bson.UnmarshalValue(bt, data, &v); err != nil {
			return err
		}
		*s = StringObjectID(v)
		return nil
	default:
		return fmt.Errorf("mongo: cannot decode bson type %v as StringObjectID", bt)
	}
}
