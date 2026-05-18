package mongo

// FetchResult bundles a paginated result set with the total count.
// Mirrors Foxel.Mongo.Models.FetchResult<T>.
type FetchResult[T any] struct {
	Count  int64 `json:"count"`
	Models []T   `json:"models"`
}
