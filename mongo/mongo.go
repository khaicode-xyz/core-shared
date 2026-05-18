// Package mongo is the core-shared MongoDB toolkit. It mirrors the
// Shareds/foxel-lib Foxel.Mongo C# package: string-based date/time
// serializers, a generic repository over a strongly-typed collection,
// a shared entity base, and small bson helpers.
//
// The connection layer is a thin wrapper around the official
// go.mongodb.org/mongo-driver/v2 client. Custom codecs registered on
// the driver registry ensure time.Time values round-trip as the
// "2006-01-02 15:04:05" string format used by the rest of the platform.
package mongo

const (
	// DateTimeLayout matches Foxel.Constants.DateConstants.DATETIME_DEFAULT_FORMAT.
	DateTimeLayout = "2006-01-02 15:04:05"
	// DateOnlyLayout matches Foxel.Constants.DateConstants.DATEONLY_DEFAULT_FORMAT.
	DateOnlyLayout = "2006-01-02"
	// TimeOnlyLayout matches Foxel.Constants.DateConstants.TIMEONLY_DEFAULT_FORMAT.
	TimeOnlyLayout = "15:04:05"
)
