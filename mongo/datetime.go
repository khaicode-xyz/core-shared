package mongo

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// DateTime is the explicit string-formatted datetime type. Mirrors the
// behavior the Foxel.Mongo BsonDateTimeSerializer applies globally to
// every System.DateTime: BSON encodes as the "2006-01-02 15:04:05"
// string in UTC and decodes from either a BSON string or a native
// BSON DateTime.
//
// Use DateTime in models when you want the round-trip to be tied to
// the type itself (rather than relying on the global registry codec
// produced by NewRegistry).
type DateTime time.Time

// Time returns the underlying time.Time.
func (d DateTime) Time() time.Time { return time.Time(d) }

// IsZero reports whether the value is the zero time.
func (d DateTime) IsZero() bool { return time.Time(d).IsZero() }

// String formats the value with DateTimeLayout.
func (d DateTime) String() string { return time.Time(d).UTC().Format(DateTimeLayout) }

// MarshalBSONValue implements bson.ValueMarshaler.
func (d DateTime) MarshalBSONValue() (byte, []byte, error) {
	t, data, err := bson.MarshalValue(d.String())
	return byte(t), data, err
}

// UnmarshalBSONValue implements bson.ValueUnmarshaler.
func (d *DateTime) UnmarshalBSONValue(typ byte, data []byte) error {
	t, err := decodeTimeValue(bson.Type(typ), data, DateTimeLayout)
	if err != nil {
		return err
	}
	*d = DateTime(t)
	return nil
}

// MarshalJSON keeps the JSON wire format identical to the DB format.
func (d DateTime) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", d.String())), nil
}

// UnmarshalJSON accepts either DateTimeLayout or RFC3339.
func (d *DateTime) UnmarshalJSON(data []byte) error {
	t, err := parseFlexibleTime(trimQuotes(data), DateTimeLayout)
	if err != nil {
		return err
	}
	*d = DateTime(t)
	return nil
}

// DateOnly stores a calendar date as "2006-01-02".
type DateOnly time.Time

// Time returns the underlying time.Time (midnight UTC).
func (d DateOnly) Time() time.Time { return time.Time(d) }

// IsZero reports whether the value is the zero time.
func (d DateOnly) IsZero() bool { return time.Time(d).IsZero() }

// String formats the value with DateOnlyLayout.
func (d DateOnly) String() string { return time.Time(d).UTC().Format(DateOnlyLayout) }

// MarshalBSONValue implements bson.ValueMarshaler.
func (d DateOnly) MarshalBSONValue() (byte, []byte, error) {
	t, data, err := bson.MarshalValue(d.String())
	return byte(t), data, err
}

// UnmarshalBSONValue implements bson.ValueUnmarshaler.
func (d *DateOnly) UnmarshalBSONValue(typ byte, data []byte) error {
	t, err := decodeTimeValue(bson.Type(typ), data, DateOnlyLayout)
	if err != nil {
		return err
	}
	*d = DateOnly(t)
	return nil
}

// MarshalJSON encodes as the DateOnlyLayout string.
func (d DateOnly) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", d.String())), nil
}

// UnmarshalJSON accepts DateOnlyLayout or RFC3339.
func (d *DateOnly) UnmarshalJSON(data []byte) error {
	t, err := parseFlexibleTime(trimQuotes(data), DateOnlyLayout)
	if err != nil {
		return err
	}
	*d = DateOnly(t)
	return nil
}

// TimeOnly stores a clock time as "15:04:05".
type TimeOnly time.Time

// Time returns the underlying time.Time (1 Jan 0001 base).
func (t TimeOnly) Time() time.Time { return time.Time(t) }

// IsZero reports whether the value is the zero time.
func (t TimeOnly) IsZero() bool { return time.Time(t).IsZero() }

// String formats the value with TimeOnlyLayout.
func (t TimeOnly) String() string { return time.Time(t).UTC().Format(TimeOnlyLayout) }

// MarshalBSONValue implements bson.ValueMarshaler.
func (t TimeOnly) MarshalBSONValue() (byte, []byte, error) {
	bt, data, err := bson.MarshalValue(t.String())
	return byte(bt), data, err
}

// UnmarshalBSONValue implements bson.ValueUnmarshaler.
func (t *TimeOnly) UnmarshalBSONValue(typ byte, data []byte) error {
	v, err := decodeTimeValue(bson.Type(typ), data, TimeOnlyLayout)
	if err != nil {
		return err
	}
	*t = TimeOnly(v)
	return nil
}

// MarshalJSON encodes as the TimeOnlyLayout string.
func (t TimeOnly) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", t.String())), nil
}

// UnmarshalJSON accepts TimeOnlyLayout.
func (t *TimeOnly) UnmarshalJSON(data []byte) error {
	v, err := parseFlexibleTime(trimQuotes(data), TimeOnlyLayout)
	if err != nil {
		return err
	}
	*t = TimeOnly(v)
	return nil
}

func decodeTimeValue(typ bson.Type, data []byte, layout string) (time.Time, error) {
	switch typ {
	case bson.TypeNull, bson.TypeUndefined:
		return time.Time{}, nil
	case bson.TypeDateTime:
		var t time.Time
		if err := bson.UnmarshalValue(typ, data, &t); err != nil {
			return time.Time{}, err
		}
		return t.UTC(), nil
	case bson.TypeString:
		var s string
		if err := bson.UnmarshalValue(typ, data, &s); err != nil {
			return time.Time{}, err
		}
		return parseFlexibleTime(s, layout)
	default:
		return time.Time{}, fmt.Errorf("mongo: cannot decode bson type %v as datetime", typ)
	}
}

func parseFlexibleTime(value, primaryLayout string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	layouts := []string{primaryLayout, time.RFC3339Nano, time.RFC3339, DateTimeLayout, DateOnlyLayout, TimeOnlyLayout}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, value, time.UTC); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("mongo: cannot parse %q as time", value)
}

func trimQuotes(data []byte) string {
	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
		return string(data[1 : len(data)-1])
	}
	return string(data)
}
