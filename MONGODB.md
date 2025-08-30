# GoLara MongoDB Integration Guide

## Overview

GoLara now uses **MongoDB** as its primary database with a custom ODM (Object Document Mapper) that provides Laravel-like query building capabilities.

## Key Changes from SQL to MongoDB

### 1. Database Connection
```go
// Before (SQL)
db, err := database.Connect("mysql", "user:password@/dbname")

// After (MongoDB)
db, err := database.Connect("mongodb://localhost:27017", "dbname")
```

### 2. Model Definition
```go
// Before (SQL)
type User struct {
    ID        uint      `json:"id" db:"id"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
    Name      string    `json:"name" db:"name"`
}

// After (MongoDB)
type User struct {
    database.Model `bson:",inline"`
    Name           string `json:"name" bson:"name"`
    Email          string `json:"email" bson:"email"`
}
```

### 3. Query Building
```go
// Before (SQL)
db.NewQueryBuilder().
    Table("users").
    Where("id", "=", 1).
    First(&user)

// After (MongoDB)
db.NewQueryBuilder().
    Collection("users").
    Where("_id", "=", objectID).
    First(&user)
```

## MongoDB ODM Features

### 1. Base Model
```go
type Model struct {
    ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
    CreatedAt time.Time          `json:"created_at" bson:"created_at"`
    UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
}
```

### 2. Query Builder Methods

#### Basic Queries
```go
// Find all documents
var users []User
err := db.NewQueryBuilder().
    Collection("users").
    Get(&users)

// Find one document
var user User
err := db.NewQueryBuilder().
    Collection("users").
    Where("email", "=", "john@example.com").
    First(&user)

// Count documents
count, err := db.NewQueryBuilder().
    Collection("users").
    Where("active", "=", true).
    Count()
```

#### Advanced Filtering
```go
// MongoDB operators
db.NewQueryBuilder().
    Collection("users").
    Where("age", ">", 18).              // $gt
    Where("age", ">=", 21).             // $gte
    Where("age", "<", 65).              // $lt
    Where("age", "<=", 60).             // $lte
    Where("status", "!=", "banned").    // $ne
    Where("name", "like", "John").      // $regex with case-insensitive
    WhereIn("role", []interface{}{"admin", "user"}). // $in
    WhereNotIn("status", []interface{}{"deleted", "banned"}). // $nin
    WhereExists("profile.avatar").      // $exists: true
    WhereNotExists("deleted_at").       // $exists: false
    Get(&users)
```

#### Sorting and Pagination
```go
// Sorting
db.NewQueryBuilder().
    Collection("posts").
    OrderBy("created_at", "DESC").
    OrderBy("title", "ASC").
    Get(&posts)

// Pagination
db.NewQueryBuilder().
    Collection("users").
    Skip(20).        // Offset equivalent
    Limit(10).
    Get(&users)
```

#### Projection (Field Selection)
```go
// Select specific fields
var users []User
err := db.NewQueryBuilder().
    Collection("users").
    Select("name", "email").
    Get(&users)
```

### 3. CRUD Operations

#### Insert
```go
// Single document
user := User{Name: "John", Email: "john@example.com"}
userID, err := db.NewQueryBuilder().
    Collection("users").
    Insert(user)

// Multiple documents
users := []interface{}{
    User{Name: "John", Email: "john@example.com"},
    User{Name: "Jane", Email: "jane@example.com"},
}
ids, err := db.NewQueryBuilder().
    Collection("users").
    InsertMany(users)
```

#### Update
```go
// Update many
result, err := db.NewQueryBuilder().
    Collection("users").
    Where("active", "=", false).
    Update(bson.M{"$set": bson.M{"status": "inactive"}})

// Update one
result, err := db.NewQueryBuilder().
    Collection("users").
    Where("_id", "=", objectID).
    UpdateOne(bson.M{"$set": bson.M{"last_login": time.Now()}})

// Replace document
newUser := User{Name: "John Updated", Email: "john.new@example.com"}
result, err := db.NewQueryBuilder().
    Collection("users").
    Where("_id", "=", objectID).
    ReplaceOne(newUser)
```

#### Delete
```go
// Delete many
result, err := db.NewQueryBuilder().
    Collection("users").
    Where("active", "=", false).
    Delete()

// Delete one
result, err := db.NewQueryBuilder().
    Collection("users").
    Where("_id", "=", objectID).
    DeleteOne()
```

### 4. Aggregation Pipeline
```go
// Complex aggregation
pipeline := []bson.M{
    {"$match": bson.M{"status": "active"}},
    {"$group": bson.M{
        "_id":   "$department",
        "count": bson.M{"$sum": 1},
        "avgAge": bson.M{"$avg": "$age"},
    }},
    {"$sort": bson.M{"count": -1}},
    {"$limit": 10},
}

var results []bson.M
err := db.NewQueryBuilder().
    Collection("users").
    Aggregate(pipeline, &results)
```

### 5. Indexes
```go
// Create index
err := db.CreateIndex("users", bson.M{
    "email": 1, // Ascending index
}, options.Index().SetUnique(true))

// Compound index
err := db.CreateIndex("posts", bson.M{
    "author_id": 1,
    "created_at": -1, // Descending
}, nil)

// Text index for search
err := db.CreateIndex("articles", bson.M{
    "title": "text",
    "content": "text",
}, nil)
```

## Model Patterns

### 1. Complete Model Example
```go
package models

import (
    "github.com/taeyelor/golara/framework/database"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "time"
)

type User struct {
    database.Model `bson:",inline"`
    Name           string             `json:"name" bson:"name"`
    Email          string             `json:"email" bson:"email"`
    Password       string             `json:"-" bson:"password"`
    Profile        UserProfile        `json:"profile" bson:"profile"`
    Roles          []string           `json:"roles" bson:"roles"`
    Settings       map[string]interface{} `json:"settings" bson:"settings"`
    LastLogin      *time.Time         `json:"last_login,omitempty" bson:"last_login,omitempty"`
}

type UserProfile struct {
    FirstName string `json:"first_name" bson:"first_name"`
    LastName  string `json:"last_name" bson:"last_name"`
    Avatar    string `json:"avatar" bson:"avatar"`
    Bio       string `json:"bio" bson:"bio"`
}

func (u *User) CollectionName() string {
    return "users"
}

// Static methods
func FindUserByEmail(db *database.DB, email string) (*User, error) {
    var user User
    err := db.NewQueryBuilder().
        Collection("users").
        Where("email", "=", email).
        First(&user)
    return &user, err
}

func GetActiveUsers(db *database.DB) ([]User, error) {
    var users []User
    err := db.NewQueryBuilder().
        Collection("users").
        WhereExists("last_login").
        OrderBy("last_login", "DESC").
        Get(&users)
    return users, err
}
```

### 2. Model Methods
```go
func (u *User) Save(db *database.DB) error {
    if u.ID.IsZero() {
        // New document
        u.SetTimestamps()
        id, err := db.NewQueryBuilder().
            Collection(u.CollectionName()).
            Insert(u)
        if err != nil {
            return err
        }
        u.ID = *id
    } else {
        // Update existing
        u.BeforeUpdate()
        _, err := db.NewQueryBuilder().
            Collection(u.CollectionName()).
            Where("_id", "=", u.ID).
            ReplaceOne(u)
        return err
    }
    return nil
}

func (u *User) Delete(db *database.DB) error {
    _, err := db.NewQueryBuilder().
        Collection(u.CollectionName()).
        Where("_id", "=", u.ID).
        DeleteOne()
    return err
}

func (u *User) AddRole(db *database.DB, role string) error {
    _, err := db.NewQueryBuilder().
        Collection(u.CollectionName()).
        Where("_id", "=", u.ID).
        UpdateOne(bson.M{"$addToSet": bson.M{"roles": role}})
    return err
}
```

## Environment Configuration

### .env File
```env
APP_NAME=MyApp
APP_ENV=local
APP_DEBUG=true
APP_PORT=:8080

# MongoDB Configuration
DB_CONNECTION=mongodb
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=myapp

# Alternative with authentication
# MONGODB_URI=mongodb://username:password@localhost:27017/myapp?authSource=admin
```

### Docker Compose for Development
```yaml
version: '3.8'
services:
  mongodb:
    image: mongo:6.0
    container_name: golara_mongodb
    restart: always
    ports:
      - "27017:27017"
    environment:
      MONGO_INITDB_ROOT_USERNAME: admin
      MONGO_INITDB_ROOT_PASSWORD: password
      MONGO_INITDB_DATABASE: golara
    volumes:
      - mongodb_data:/data/db

volumes:
  mongodb_data:
```

## Migration from SQL

### 1. Update Dependencies
```bash
go mod tidy
go get go.mongodb.org/mongo-driver/mongo
```

### 2. Update Models
- Change `database.Model` embedding
- Use `bson` tags instead of `db` tags
- Use `primitive.ObjectID` for IDs
- Replace `Table()` with `Collection()`

### 3. Update Queries
- Replace table names with collection names
- Use MongoDB operators (`$gt`, `$in`, etc.)
- Update field references (e.g., `id` → `_id`)

### 4. Handle ObjectIDs
```go
// Convert string to ObjectID
objectID, err := primitive.ObjectIDFromHex(idString)

// Convert ObjectID to string
idString := objectID.Hex()
```

This MongoDB integration provides a powerful, flexible foundation for building modern web applications with GoLara while maintaining the familiar Laravel-like developer experience.
