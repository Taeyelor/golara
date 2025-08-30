package database

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// DB represents a database connection
type DB struct {
	*sql.DB
	driver string
}

// Model represents a base model with common fields
type Model struct {
	ID        uint      `json:"id" db:"id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// QueryBuilder provides a fluent interface for building queries
type QueryBuilder struct {
	db          *DB
	table       string
	selectCols  []string
	whereConds  []string
	whereValues []interface{}
	orderBy     []string
	limitCount  int
	offsetCount int
	joins       []string
}

// Connect creates a new database connection
func Connect(driver, dsn string) (*DB, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &DB{DB: db, driver: driver}, nil
}

// NewQueryBuilder creates a new query builder
func (db *DB) NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		db:          db,
		selectCols:  []string{"*"},
		whereConds:  []string{},
		whereValues: []interface{}{},
		orderBy:     []string{},
		joins:       []string{},
	}
}

// Table sets the table name
func (qb *QueryBuilder) Table(table string) *QueryBuilder {
	qb.table = table
	return qb
}

// Select sets the columns to select
func (qb *QueryBuilder) Select(columns ...string) *QueryBuilder {
	qb.selectCols = columns
	return qb
}

// Where adds a WHERE condition
func (qb *QueryBuilder) Where(column, operator string, value interface{}) *QueryBuilder {
	qb.whereConds = append(qb.whereConds, fmt.Sprintf("%s %s ?", column, operator))
	qb.whereValues = append(qb.whereValues, value)
	return qb
}

// WhereIn adds a WHERE IN condition
func (qb *QueryBuilder) WhereIn(column string, values []interface{}) *QueryBuilder {
	placeholders := strings.Repeat("?,", len(values))
	placeholders = placeholders[:len(placeholders)-1] // Remove trailing comma

	qb.whereConds = append(qb.whereConds, fmt.Sprintf("%s IN (%s)", column, placeholders))
	qb.whereValues = append(qb.whereValues, values...)
	return qb
}

// OrderBy adds an ORDER BY clause
func (qb *QueryBuilder) OrderBy(column, direction string) *QueryBuilder {
	qb.orderBy = append(qb.orderBy, fmt.Sprintf("%s %s", column, direction))
	return qb
}

// Limit sets the LIMIT
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.limitCount = limit
	return qb
}

// Offset sets the OFFSET
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.offsetCount = offset
	return qb
}

// Join adds a JOIN clause
func (qb *QueryBuilder) Join(table, condition string) *QueryBuilder {
	qb.joins = append(qb.joins, fmt.Sprintf("JOIN %s ON %s", table, condition))
	return qb
}

// LeftJoin adds a LEFT JOIN clause
func (qb *QueryBuilder) LeftJoin(table, condition string) *QueryBuilder {
	qb.joins = append(qb.joins, fmt.Sprintf("LEFT JOIN %s ON %s", table, condition))
	return qb
}

// Get executes the query and returns results
func (qb *QueryBuilder) Get(dest interface{}) error {
	query := qb.buildSelectQuery()
	rows, err := qb.db.Query(query, qb.whereValues...)
	if err != nil {
		return err
	}
	defer rows.Close()

	return qb.scanRows(rows, dest)
}

// First executes the query and returns the first result
func (qb *QueryBuilder) First(dest interface{}) error {
	qb.Limit(1)
	query := qb.buildSelectQuery()

	row := qb.db.QueryRow(query, qb.whereValues...)
	return qb.scanRow(row, dest)
}

// Count returns the count of matching records
func (qb *QueryBuilder) Count() (int, error) {
	originalCols := qb.selectCols
	qb.selectCols = []string{"COUNT(*)"}

	query := qb.buildSelectQuery()
	qb.selectCols = originalCols // Restore original columns

	var count int
	err := qb.db.QueryRow(query, qb.whereValues...).Scan(&count)
	return count, err
}

// Insert inserts a new record
func (qb *QueryBuilder) Insert(data map[string]interface{}) (int64, error) {
	columns := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))
	placeholders := make([]string, 0, len(data))

	for column, value := range data {
		columns = append(columns, column)
		values = append(values, value)
		placeholders = append(placeholders, "?")
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		qb.table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	result, err := qb.db.Exec(query, values...)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// Update updates existing records
func (qb *QueryBuilder) Update(data map[string]interface{}) (int64, error) {
	setParts := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for column, value := range data {
		setParts = append(setParts, fmt.Sprintf("%s = ?", column))
		values = append(values, value)
	}

	// Add WHERE values after SET values
	values = append(values, qb.whereValues...)

	query := fmt.Sprintf("UPDATE %s SET %s", qb.table, strings.Join(setParts, ", "))

	if len(qb.whereConds) > 0 {
		query += " WHERE " + strings.Join(qb.whereConds, " AND ")
	}

	result, err := qb.db.Exec(query, values...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// Delete deletes records
func (qb *QueryBuilder) Delete() (int64, error) {
	query := fmt.Sprintf("DELETE FROM %s", qb.table)

	if len(qb.whereConds) > 0 {
		query += " WHERE " + strings.Join(qb.whereConds, " AND ")
	}

	result, err := qb.db.Exec(query, qb.whereValues...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// buildSelectQuery builds the SELECT query
func (qb *QueryBuilder) buildSelectQuery() string {
	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(qb.selectCols, ", "), qb.table)

	// Add JOINs
	if len(qb.joins) > 0 {
		query += " " + strings.Join(qb.joins, " ")
	}

	// Add WHERE
	if len(qb.whereConds) > 0 {
		query += " WHERE " + strings.Join(qb.whereConds, " AND ")
	}

	// Add ORDER BY
	if len(qb.orderBy) > 0 {
		query += " ORDER BY " + strings.Join(qb.orderBy, ", ")
	}

	// Add LIMIT and OFFSET
	if qb.limitCount > 0 {
		query += fmt.Sprintf(" LIMIT %d", qb.limitCount)
	}

	if qb.offsetCount > 0 {
		query += fmt.Sprintf(" OFFSET %d", qb.offsetCount)
	}

	return query
}

// scanRows scans multiple rows into a slice
func (qb *QueryBuilder) scanRows(rows *sql.Rows, dest interface{}) error {
	destValue := reflect.ValueOf(dest).Elem()
	destType := destValue.Type().Elem()

	for rows.Next() {
		item := reflect.New(destType).Elem()
		if err := qb.scanRowIntoStruct(rows, item.Addr().Interface()); err != nil {
			return err
		}
		destValue.Set(reflect.Append(destValue, item))
	}

	return rows.Err()
}

// scanRow scans a single row
func (qb *QueryBuilder) scanRow(row *sql.Row, dest interface{}) error {
	return qb.scanRowIntoStruct(row, dest)
}

// scanRowIntoStruct scans a row into a struct
func (qb *QueryBuilder) scanRowIntoStruct(scanner interface{}, dest interface{}) error {
	destValue := reflect.ValueOf(dest).Elem()
	destType := destValue.Type()

	// Get field pointers for scanning
	fieldPtrs := make([]interface{}, destType.NumField())
	for i := 0; i < destType.NumField(); i++ {
		fieldPtrs[i] = destValue.Field(i).Addr().Interface()
	}

	switch s := scanner.(type) {
	case *sql.Rows:
		return s.Scan(fieldPtrs...)
	case *sql.Row:
		return s.Scan(fieldPtrs...)
	default:
		return fmt.Errorf("unsupported scanner type")
	}
}
