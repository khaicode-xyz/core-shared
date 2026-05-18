package mongo

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// NewRegistry returns a bson Registry pre-configured to encode/decode
// time.Time as a "2006-01-02 15:04:05" UTC string. It matches the
// behavior of Foxel.Mongo's BsonDateTimeSerializer registered at
// process startup.
//
// Pass the returned registry to options.Client().SetRegistry(...) so
// every collection opened on the client uses the same wire format.
func NewRegistry() *bson.Registry {
	reg := bson.NewRegistry()
	timeType := reflect.TypeOf(time.Time{})
	reg.RegisterTypeEncoder(timeType, bson.ValueEncoderFunc(encodeTimeAsString))
	reg.RegisterTypeDecoder(timeType, bson.ValueDecoderFunc(decodeTimeFromString))
	return reg
}

func encodeTimeAsString(_ bson.EncodeContext, vw bson.ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != reflect.TypeOf(time.Time{}) {
		return errors.New("mongo: encoder received non time.Time value")
	}
	t := val.Interface().(time.Time)
	return vw.WriteString(t.UTC().Format(DateTimeLayout))
}

func decodeTimeFromString(_ bson.DecodeContext, vr bson.ValueReader, val reflect.Value) error {
	if !val.CanSet() {
		return errors.New("mongo: decoder received non-settable time.Time value")
	}
	switch vr.Type() {
	case bson.TypeNull, bson.TypeUndefined:
		val.Set(reflect.ValueOf(time.Time{}))
		return vr.ReadNull()
	case bson.TypeDateTime:
		ms, err := vr.ReadDateTime()
		if err != nil {
			return err
		}
		val.Set(reflect.ValueOf(time.UnixMilli(ms).UTC()))
		return nil
	case bson.TypeString:
		s, err := vr.ReadString()
		if err != nil {
			return err
		}
		if s == "" {
			val.Set(reflect.ValueOf(time.Time{}))
			return nil
		}
		t, err := parseFlexibleTime(s, DateTimeLayout)
		if err != nil {
			return err
		}
		val.Set(reflect.ValueOf(t))
		return nil
	default:
		return fmt.Errorf("mongo: cannot decode bson type %v as time.Time", vr.Type())
	}
}
