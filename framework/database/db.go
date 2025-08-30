package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// DB represents a MongoDB database connection
type DB struct {
	Client   *mongo.Client
	Database *mongo.Database
	Name     string
}

// Model represents a base model with common fields for MongoDB
type Model struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
}

// QueryBuilder provides a fluent interface for building MongoDB queries
type QueryBuilder struct {
	db         *DB
	collection string
	filter     bson.M
	sort       bson.D
	limit      int64
	skip       int64
	projection bson.M
	ctx        context.Context
}

// Connect creates a new MongoDB connection
func Connect(uri, dbName string) (*DB, error) {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	// Ping the database to verify connection
	if err := client.Ping(context.TODO(), nil); err != nil {
		return nil, err
	}

	database := client.Database(dbName)

	return &DB{
		Client:   client,
		Database: database,
		Name:     dbName,
	}, nil
}

// NewQueryBuilder creates a new query builder
func (db *DB) NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		db:         db,
		filter:     bson.M{},
		sort:       bson.D{},
		projection: bson.M{},
		ctx:        context.Background(),
	}
}

// Collection sets the collection name
func (qb *QueryBuilder) Collection(collection string) *QueryBuilder {
	qb.collection = collection
	return qb
}

// Where adds a filter condition
func (qb *QueryBuilder) Where(field string, operator string, value interface{}) *QueryBuilder {
	switch operator {
	case "=", "==":
		qb.filter[field] = value
	case "!=", "<>":
		qb.filter[field] = bson.M{"$ne": value}
	case ">":
		qb.filter[field] = bson.M{"$gt": value}
	case ">=":
		qb.filter[field] = bson.M{"$gte": value}
	case "<":
		qb.filter[field] = bson.M{"$lt": value}
	case "<=":
		qb.filter[field] = bson.M{"$lte": value}
	case "like":
		qb.filter[field] = bson.M{"$regex": value, "$options": "i"}
	case "in":
		if arr, ok := value.([]interface{}); ok {
			qb.filter[field] = bson.M{"$in": arr}
		}
	case "nin":
		if arr, ok := value.([]interface{}); ok {
			qb.filter[field] = bson.M{"$nin": arr}
		}
	default:
		qb.filter[field] = value
	}
	return qb
}

// WhereIn adds an $in filter condition
func (qb *QueryBuilder) WhereIn(field string, values []interface{}) *QueryBuilder {
	qb.filter[field] = bson.M{"$in": values}
	return qb
}

// WhereNotIn adds a $nin filter condition
func (qb *QueryBuilder) WhereNotIn(field string, values []interface{}) *QueryBuilder {
	qb.filter[field] = bson.M{"$nin": values}
	return qb
}

// WhereExists checks if a field exists
func (qb *QueryBuilder) WhereExists(field string) *QueryBuilder {
	qb.filter[field] = bson.M{"$exists": true}
	return qb
}

// WhereNotExists checks if a field doesn't exist
func (qb *QueryBuilder) WhereNotExists(field string) *QueryBuilder {
	qb.filter[field] = bson.M{"$exists": false}
	return qb
}

// OrderBy adds sorting
func (qb *QueryBuilder) OrderBy(field string, direction string) *QueryBuilder {
	var order int32 = 1
	if direction == "desc" || direction == "DESC" {
		order = -1
	}
	qb.sort = append(qb.sort, bson.E{Key: field, Value: order})
	return qb
}

// Limit sets the limit
func (qb *QueryBuilder) Limit(limit int64) *QueryBuilder {
	qb.limit = limit
	return qb
}

// Skip sets the skip (offset equivalent)
func (qb *QueryBuilder) Skip(skip int64) *QueryBuilder {
	qb.skip = skip
	return qb
}

// Offset sets the skip (alias for Skip)
func (qb *QueryBuilder) Offset(offset int64) *QueryBuilder {
	return qb.Skip(offset)
}

// Select sets projection (fields to include)
func (qb *QueryBuilder) Select(fields ...string) *QueryBuilder {
	for _, field := range fields {
		qb.projection[field] = 1
	}
	return qb
}

// Context sets the context for the query
func (qb *QueryBuilder) Context(ctx context.Context) *QueryBuilder {
	qb.ctx = ctx
	return qb
}

// Get executes the query and returns multiple documents
func (qb *QueryBuilder) Get(dest interface{}) error {
	coll := qb.db.Database.Collection(qb.collection)

	opts := options.Find()

	if len(qb.sort) > 0 {
		opts.SetSort(qb.sort)
	}
	if qb.limit > 0 {
		opts.SetLimit(qb.limit)
	}
	if qb.skip > 0 {
		opts.SetSkip(qb.skip)
	}
	if len(qb.projection) > 0 {
		opts.SetProjection(qb.projection)
	}

	cursor, err := coll.Find(qb.ctx, qb.filter, opts)
	if err != nil {
		return err
	}
	defer cursor.Close(qb.ctx)

	return cursor.All(qb.ctx, dest)
}

// First executes the query and returns the first document
func (qb *QueryBuilder) First(dest interface{}) error {
	coll := qb.db.Database.Collection(qb.collection)

	opts := options.FindOne()

	if len(qb.sort) > 0 {
		opts.SetSort(qb.sort)
	}
	if len(qb.projection) > 0 {
		opts.SetProjection(qb.projection)
	}

	result := coll.FindOne(qb.ctx, qb.filter, opts)

	return result.Decode(dest)
}

// Count returns the count of matching documents
func (qb *QueryBuilder) Count() (int64, error) {
	coll := qb.db.Database.Collection(qb.collection)

	return coll.CountDocuments(qb.ctx, qb.filter)
}

// Insert inserts a new document
func (qb *QueryBuilder) Insert(document interface{}) (*primitive.ObjectID, error) {
	coll := qb.db.Database.Collection(qb.collection)

	// Set timestamps if it's a model
	if model, ok := document.(interface{ SetTimestamps() }); ok {
		model.SetTimestamps()
	}

	result, err := coll.InsertOne(qb.ctx, document)
	if err != nil {
		return nil, err
	}

	if objectID, ok := result.InsertedID.(primitive.ObjectID); ok {
		return &objectID, nil
	}

	return nil, fmt.Errorf("failed to get inserted ID")
}

// InsertMany inserts multiple documents
func (qb *QueryBuilder) InsertMany(documents []interface{}) ([]primitive.ObjectID, error) {
	coll := qb.db.Database.Collection(qb.collection)

	// Set timestamps for models
	for _, doc := range documents {
		if model, ok := doc.(interface{ SetTimestamps() }); ok {
			model.SetTimestamps()
		}
	}

	result, err := coll.InsertMany(qb.ctx, documents)
	if err != nil {
		return nil, err
	}

	var ids []primitive.ObjectID
	for _, id := range result.InsertedIDs {
		if objectID, ok := id.(primitive.ObjectID); ok {
			ids = append(ids, objectID)
		}
	}

	return ids, nil
}

// Update updates existing documents
func (qb *QueryBuilder) Update(update bson.M) (*mongo.UpdateResult, error) {
	coll := qb.db.Database.Collection(qb.collection)

	// Add updated_at timestamp
	if update["$set"] == nil {
		update["$set"] = bson.M{}
	}
	if setFields, ok := update["$set"].(bson.M); ok {
		setFields["updated_at"] = time.Now()
	}

	return coll.UpdateMany(qb.ctx, qb.filter, update)
}

// UpdateOne updates a single document
func (qb *QueryBuilder) UpdateOne(update bson.M) (*mongo.UpdateResult, error) {
	coll := qb.db.Database.Collection(qb.collection)

	// Add updated_at timestamp
	if update["$set"] == nil {
		update["$set"] = bson.M{}
	}
	if setFields, ok := update["$set"].(bson.M); ok {
		setFields["updated_at"] = time.Now()
	}

	return coll.UpdateOne(qb.ctx, qb.filter, update)
}

// ReplaceOne replaces a single document
func (qb *QueryBuilder) ReplaceOne(replacement interface{}) (*mongo.UpdateResult, error) {
	coll := qb.db.Database.Collection(qb.collection)

	// Set timestamps if it's a model
	if model, ok := replacement.(interface{ SetTimestamps() }); ok {
		model.SetTimestamps()
	}

	return coll.ReplaceOne(qb.ctx, qb.filter, replacement)
}

// Delete deletes documents
func (qb *QueryBuilder) Delete() (*mongo.DeleteResult, error) {
	coll := qb.db.Database.Collection(qb.collection)

	return coll.DeleteMany(qb.ctx, qb.filter)
}

// DeleteOne deletes a single document
func (qb *QueryBuilder) DeleteOne() (*mongo.DeleteResult, error) {
	coll := qb.db.Database.Collection(qb.collection)

	return coll.DeleteOne(qb.ctx, qb.filter)
}

// Aggregate performs aggregation pipeline
func (qb *QueryBuilder) Aggregate(pipeline []bson.M, dest interface{}) error {
	coll := qb.db.Database.Collection(qb.collection)

	cursor, err := coll.Aggregate(qb.ctx, pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(qb.ctx)

	return cursor.All(qb.ctx, dest)
}

// SetTimestamps sets created_at and updated_at for the model
func (m *Model) SetTimestamps() {
	now := time.Now()
	if m.ID.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = now
}

// BeforeInsert hook called before inserting
func (m *Model) BeforeInsert() {
	now := time.Now()
	m.CreatedAt = now
	m.UpdatedAt = now
}

// BeforeUpdate hook called before updating
func (m *Model) BeforeUpdate() {
	m.UpdatedAt = time.Now()
}

// Collection returns the MongoDB collection
func (db *DB) Collection(name string) *mongo.Collection {
	return db.Database.Collection(name)
}

// Disconnect closes the MongoDB connection
func (db *DB) Disconnect() error {
	return db.Client.Disconnect(context.TODO())
}

// Ping checks the MongoDB connection
func (db *DB) Ping() error {
	return db.Client.Ping(context.TODO(), nil)
}

// CreateIndex creates an index on the specified collection
func (db *DB) CreateIndex(collection string, keys bson.M, options *options.IndexOptions) error {
	coll := db.Database.Collection(collection)

	indexModel := mongo.IndexModel{
		Keys:    keys,
		Options: options,
	}

	_, err := coll.Indexes().CreateOne(context.TODO(), indexModel)
	return err
}

// DropIndex drops an index from the specified collection
func (db *DB) DropIndex(collection, indexName string) error {
	coll := db.Database.Collection(collection)

	_, err := coll.Indexes().DropOne(context.TODO(), indexName)
	return err
}
