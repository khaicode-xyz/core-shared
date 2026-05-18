package mongo

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// IndexSpec describes a single MongoDB index. Mirrors the
// (name, definition, sparse) triple used by Foxel.Mongo's CreateIndexes.
type IndexSpec struct {
	Name   string
	Keys   any
	Sparse bool
	Unique bool
}

// Repository is the generic CRUD layer mirroring
// Foxel.Mongo.Repositories.MongoRepository<T>. The type parameter T is
// the document model; T must implement Document (Entity satisfies it
// when embedded).
type Repository[T Document] struct {
	client     *Client
	collection string
}

// NewRepository constructs a Repository for the given model type. The
// collection name is derived from CollectionNamer or the type name
// (snake_case).
func NewRepository[T Document](client *Client) *Repository[T] {
	return &Repository[T]{client: client, collection: ResolveCollectionName[T]()}
}

// NewRepositoryWithName lets the caller override the collection name.
func NewRepositoryWithName[T Document](client *Client, name string) *Repository[T] {
	return &Repository[T]{client: client, collection: name}
}

// CollectionName returns the resolved collection name.
func (r *Repository[T]) CollectionName() string { return r.collection }

// Collection returns the underlying *mongo.Collection.
func (r *Repository[T]) Collection(ctx context.Context) (*mongo.Collection, error) {
	return r.client.Collection(ctx, r.collection)
}

// ---------- queries ----------

// Find returns all documents matching the filter.
func (r *Repository[T]) Find(ctx context.Context, filter any, sort any) ([]T, error) {
	cursor, err := r.findCursor(ctx, filter, sort, 0, 0)
	if err != nil {
		return nil, err
	}
	return drainCursor[T](ctx, cursor)
}

// FindPaging returns a page of documents (1-based page index).
func (r *Repository[T]) FindPaging(ctx context.Context, page, size int, filter any, sort any) ([]T, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	cursor, err := r.findCursor(ctx, filter, sort, int64((page-1)*size), int64(size))
	if err != nil {
		return nil, err
	}
	return drainCursor[T](ctx, cursor)
}

// FindOne returns the first document matching the filter, or nil when no
// document matches.
func (r *Repository[T]) FindOne(ctx context.Context, filter any, sort any) (*T, error) {
	coll, err := r.Collection(ctx)
	if err != nil {
		return nil, err
	}
	opts := options.FindOne()
	if sort != nil {
		opts = opts.SetSort(sort)
	}
	var result T
	err = coll.FindOne(ctx, ensureFilter(filter), opts).Decode(&result)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Count returns the number of documents matching the filter.
func (r *Repository[T]) Count(ctx context.Context, filter any) (int64, error) {
	coll, err := r.Collection(ctx)
	if err != nil {
		return 0, err
	}
	return coll.CountDocuments(ctx, ensureFilter(filter))
}

// EstimatedCount returns the collection size using metadata (cheap).
func (r *Repository[T]) EstimatedCount(ctx context.Context) (int64, error) {
	coll, err := r.Collection(ctx)
	if err != nil {
		return 0, err
	}
	return coll.EstimatedDocumentCount(ctx)
}

// Distinct returns the distinct values of a field.
func (r *Repository[T]) Distinct(ctx context.Context, field string, filter any) ([]any, error) {
	coll, err := r.Collection(ctx)
	if err != nil {
		return nil, err
	}
	res := coll.Distinct(ctx, field, ensureFilter(filter))
	if res.Err() != nil {
		return nil, res.Err()
	}
	var values []any
	if err := res.Decode(&values); err != nil {
		return nil, err
	}
	return values, nil
}

// ---------- mutations ----------

// InsertOne inserts a document.
func (r *Repository[T]) InsertOne(ctx context.Context, doc T) error {
	coll, err := r.Collection(ctx)
	if err != nil {
		return err
	}
	_, err = coll.InsertOne(ctx, doc)
	return err
}

// InsertMany inserts multiple documents.
func (r *Repository[T]) InsertMany(ctx context.Context, docs []T) error {
	if len(docs) == 0 {
		return nil
	}
	coll, err := r.Collection(ctx)
	if err != nil {
		return err
	}
	payload := make([]any, len(docs))
	for i := range docs {
		payload[i] = docs[i]
	}
	_, err = coll.InsertMany(ctx, payload)
	return err
}

// UpdateOne applies a $set/$inc style update to a single document.
func (r *Repository[T]) UpdateOne(ctx context.Context, filter any, update any) (int64, error) {
	coll, err := r.Collection(ctx)
	if err != nil {
		return 0, err
	}
	res, err := coll.UpdateOne(ctx, ensureFilter(filter), update)
	if err != nil {
		return 0, err
	}
	return res.ModifiedCount, nil
}

// UpsertOne updates or inserts a single document.
func (r *Repository[T]) UpsertOne(ctx context.Context, filter any, update any) error {
	coll, err := r.Collection(ctx)
	if err != nil {
		return err
	}
	_, err = coll.UpdateOne(ctx, ensureFilter(filter), update, options.UpdateOne().SetUpsert(true))
	return err
}

// UpdateMany updates every document matching the filter.
func (r *Repository[T]) UpdateMany(ctx context.Context, filter any, update any) (int64, error) {
	coll, err := r.Collection(ctx)
	if err != nil {
		return 0, err
	}
	res, err := coll.UpdateMany(ctx, ensureFilter(filter), update)
	if err != nil {
		return 0, err
	}
	return res.ModifiedCount, nil
}

// UpsertMany updates or inserts every document matching the filter.
func (r *Repository[T]) UpsertMany(ctx context.Context, filter any, update any) error {
	coll, err := r.Collection(ctx)
	if err != nil {
		return err
	}
	_, err = coll.UpdateMany(ctx, ensureFilter(filter), update, options.UpdateMany().SetUpsert(true))
	return err
}

// FindOneAndUpdate atomically updates a document and returns the new (or
// previous) version. Returns nil when no document matches.
func (r *Repository[T]) FindOneAndUpdate(ctx context.Context, filter any, update any, returnAfter bool) (*T, error) {
	coll, err := r.Collection(ctx)
	if err != nil {
		return nil, err
	}
	rd := options.Before
	if returnAfter {
		rd = options.After
	}
	var result T
	err = coll.FindOneAndUpdate(ctx, ensureFilter(filter), update,
		options.FindOneAndUpdate().SetReturnDocument(rd),
	).Decode(&result)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// FindOneAndUpsert atomically upserts a document and returns it.
func (r *Repository[T]) FindOneAndUpsert(ctx context.Context, filter any, update any, returnAfter bool) (*T, error) {
	coll, err := r.Collection(ctx)
	if err != nil {
		return nil, err
	}
	rd := options.Before
	if returnAfter {
		rd = options.After
	}
	var result T
	err = coll.FindOneAndUpdate(ctx, ensureFilter(filter), update,
		options.FindOneAndUpdate().SetReturnDocument(rd).SetUpsert(true),
	).Decode(&result)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// FindOneAndDelete atomically removes a document and returns it.
func (r *Repository[T]) FindOneAndDelete(ctx context.Context, filter any) (*T, error) {
	coll, err := r.Collection(ctx)
	if err != nil {
		return nil, err
	}
	var result T
	err = coll.FindOneAndDelete(ctx, ensureFilter(filter)).Decode(&result)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ReplaceByID replaces the document whose `_id` equals doc.GetID().
func (r *Repository[T]) ReplaceByID(ctx context.Context, doc T) error {
	id := doc.GetID()
	if id == "" {
		return errors.New("mongo: document id is empty")
	}
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	coll, err := r.Collection(ctx)
	if err != nil {
		return err
	}
	_, err = coll.ReplaceOne(ctx, bson.M{"_id": oid}, doc)
	return err
}

// ReplaceByCode replaces the document whose `code` equals doc.GetCode().
func (r *Repository[T]) ReplaceByCode(ctx context.Context, doc T) error {
	code := doc.GetCode()
	if code == "" {
		return errors.New("mongo: document code is empty")
	}
	coll, err := r.Collection(ctx)
	if err != nil {
		return err
	}
	_, err = coll.ReplaceOne(ctx, bson.M{"code": code}, doc)
	return err
}

// DeleteOne deletes a single document matching the filter.
func (r *Repository[T]) DeleteOne(ctx context.Context, filter any) (int64, error) {
	coll, err := r.Collection(ctx)
	if err != nil {
		return 0, err
	}
	res, err := coll.DeleteOne(ctx, ensureFilter(filter))
	if err != nil {
		return 0, err
	}
	return res.DeletedCount, nil
}

// DeleteMany deletes every document matching the filter.
func (r *Repository[T]) DeleteMany(ctx context.Context, filter any) (int64, error) {
	coll, err := r.Collection(ctx)
	if err != nil {
		return 0, err
	}
	res, err := coll.DeleteMany(ctx, ensureFilter(filter))
	if err != nil {
		return 0, err
	}
	return res.DeletedCount, nil
}

// BulkWrite runs a batch of write operations.
func (r *Repository[T]) BulkWrite(ctx context.Context, models []mongo.WriteModel) (*mongo.BulkWriteResult, error) {
	if len(models) == 0 {
		return &mongo.BulkWriteResult{}, nil
	}
	coll, err := r.Collection(ctx)
	if err != nil {
		return nil, err
	}
	return coll.BulkWrite(ctx, models)
}

// CreateIndex creates a single index.
func (r *Repository[T]) CreateIndex(ctx context.Context, spec IndexSpec) error {
	coll, err := r.Collection(ctx)
	if err != nil {
		return err
	}
	model := mongo.IndexModel{
		Keys:    spec.Keys,
		Options: options.Index().SetName(spec.Name).SetSparse(spec.Sparse).SetUnique(spec.Unique),
	}
	_, err = coll.Indexes().CreateOne(ctx, model)
	return err
}

// CreateIndexes creates multiple indexes.
func (r *Repository[T]) CreateIndexes(ctx context.Context, specs []IndexSpec) error {
	if len(specs) == 0 {
		return nil
	}
	coll, err := r.Collection(ctx)
	if err != nil {
		return err
	}
	models := make([]mongo.IndexModel, len(specs))
	for i, spec := range specs {
		models[i] = mongo.IndexModel{
			Keys:    spec.Keys,
			Options: options.Index().SetName(spec.Name).SetSparse(spec.Sparse).SetUnique(spec.Unique),
		}
	}
	_, err = coll.Indexes().CreateMany(ctx, models)
	return err
}

// ---------- helpers ----------

func (r *Repository[T]) findCursor(ctx context.Context, filter any, sort any, skip, limit int64) (*mongo.Cursor, error) {
	coll, err := r.Collection(ctx)
	if err != nil {
		return nil, err
	}
	opts := options.Find().SetAllowDiskUse(true)
	if sort != nil {
		opts = opts.SetSort(sort)
	}
	if skip > 0 {
		opts = opts.SetSkip(skip)
	}
	if limit > 0 {
		opts = opts.SetLimit(limit)
	}
	return coll.Find(ctx, ensureFilter(filter), opts)
}

func drainCursor[T any](ctx context.Context, cursor *mongo.Cursor) ([]T, error) {
	defer cursor.Close(ctx)
	var out []T
	if err := cursor.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func ensureFilter(filter any) any {
	if filter == nil {
		return bson.M{}
	}
	return filter
}
