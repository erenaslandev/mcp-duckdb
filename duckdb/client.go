package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/marcboeker/go-duckdb/v2"
)

type Client interface {
	QueryData(ctx context.Context, query string, limit int) (QueryResult, error)
	GetConnection() *sql.Conn
	Close() error
}

type ColumnInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Position int    `json:"position"`
	IsArray  bool   `json:"is_array,omitempty"`
	IsNested bool   `json:"is_nested,omitempty"`
}

type QueryResult struct {
	Columns []ColumnInfo     `json:"columns"`
	Rows    []map[string]any `json:"rows"`
}

type DefaultClient struct {
	conn *sql.Conn
}

type Config struct {
	DSN string
}

func NewClient(cfg Config) (Client, error) {
	db, err := sql.Open("duckdb", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("can not get database: %w", err)
	}

	ctx := context.Background()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("can not verify database: %w", err)
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("can not get connection: %w", err)
	}

	err = conn.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("can not verify connection: %w", err)
	}

	return &DefaultClient{conn: conn}, nil
}

func (c *DefaultClient) QueryData(ctx context.Context, query string, limit int) (QueryResult, error) {
	cleanQuery := normalizeQuery(query)

	limitedQuery := cleanQuery
	if limit > 0 {
		if !containsLimitClause(cleanQuery) {
			limitedQuery = fmt.Sprintf("%s LIMIT %d", cleanQuery, limit)
		}
	}

	if err := c.ensureConnection(ctx); err != nil {
		return QueryResult{}, err
	}

	rows, err := c.conn.QueryContext(ctx, limitedQuery)
	if err != nil {
		return QueryResult{}, fmt.Errorf("can not execute query: %w", err)
	}
	defer rows.Close()

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return QueryResult{}, fmt.Errorf("can not get column types: %w", err)
	}

	columnNames, err := rows.Columns()
	if err != nil {
		return QueryResult{}, fmt.Errorf("can not get columns: %w", err)
	}

	var columns []ColumnInfo
	for i, ct := range columnTypes {
		dbType := ct.DatabaseTypeName()
		isArray := IsArrayType(dbType)
		isNested := len(dbType) >= 7 && dbType[:6] == "Nested"

		columns = append(columns, ColumnInfo{
			Name:     ct.Name(),
			Type:     dbType,
			Position: i + 1,
			IsArray:  isArray,
			IsNested: isNested,
		})
	}

	var results []map[string]any

	destPointers := make([]any, len(columnNames))
	stringVars := make([]string, len(columnNames))
	intVars := make([]int64, len(columnNames))
	uintVars := make([]uint64, len(columnNames))
	floatVars := make([]float64, len(columnNames))
	boolVars := make([]bool, len(columnNames))
	timeVars := make([]time.Time, len(columnNames))
	anyVars := make([]any, len(columnNames))

	for i, col := range columns {
		switch col.Type {
		case "String":
			destPointers[i] = &stringVars[i]
		case "UInt8", "UInt16", "UInt32", "UInt64":
			destPointers[i] = &uintVars[i]
		case "Int8", "Int16", "Int32", "Int64":
			destPointers[i] = &intVars[i]
		case "Float32", "Float64":
			destPointers[i] = &floatVars[i]
		case "Bool":
			destPointers[i] = &boolVars[i]
		case "Date", "DateTime":
			destPointers[i] = &timeVars[i]
		default:
			destPointers[i] = &anyVars[i]
		}
	}

	for rows.Next() {
		if err := rows.Scan(destPointers...); err != nil {
			return QueryResult{}, fmt.Errorf("can not scan data: %w", err)
		}

		row := make(map[string]any)

		for i, col := range columns {
			switch col.Type {
			case "String":
				row[col.Name] = stringVars[i]
			case "UInt8", "UInt16", "UInt32", "UInt64":
				row[col.Name] = uintVars[i]
			case "Int8", "Int16", "Int32", "Int64":
				row[col.Name] = intVars[i]
			case "Float32", "Float64":
				row[col.Name] = floatVars[i]
			case "Bool":
				row[col.Name] = boolVars[i]
			case "Date", "DateTime":
				row[col.Name] = timeVars[i].Format(time.RFC3339)
			default:
				v := anyVars[i]

				switch val := v.(type) {
				case []byte:
					row[col.Name] = string(val)
				case []any:
					if col.IsArray && len(val) > 0 {
						if _, ok := val[0].([]byte); ok {
							strArray := make([]string, len(val))
							for j, item := range val {
								if byteItem, ok := item.([]byte); ok {
									strArray[j] = string(byteItem)
								} else {
									strArray[j] = fmt.Sprint(item)
								}
							}
							row[col.Name] = strArray
						} else {
							row[col.Name] = val
						}
					} else {
						row[col.Name] = val
					}
				default:
					row[col.Name] = v
				}
			}
		}

		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return QueryResult{}, fmt.Errorf("can not process results: %w", err)
	}

	return QueryResult{
		Columns: columns,
		Rows:    results,
	}, nil
}

func (c *DefaultClient) ensureConnection(ctx context.Context) error {
	err := c.conn.PingContext(ctx)
	if err != nil {
		return fmt.Errorf("can not verify connection: %w", err)
	}
	return nil
}

func normalizeQuery(query string) string {
	query = strings.TrimSpace(query)
	if len(query) > 0 && query[len(query)-1] == ';' {
		query = query[:len(query)-1]
	}

	query = strings.TrimSpace(query)

	return query
}

func containsLimitClause(query string) bool {
	queryWithoutComments := removeComments(query)
	upperQuery := strings.ToUpper(queryWithoutComments)
	return strings.Contains(upperQuery, " LIMIT ")
}

func removeComments(query string) string {
	result := query
	for {
		startIdx := strings.Index(result, "/*")
		if startIdx == -1 {
			break
		}

		endIdx := strings.Index(result[startIdx:], "*/")
		if endIdx == -1 {
			result = result[:startIdx]
			break
		}

		endIdx = startIdx + endIdx + 2
		result = result[:startIdx] + " " + result[endIdx:]
	}

	lines := strings.Split(result, "\n")
	for i, line := range lines {
		commentIdx := strings.Index(line, "--")
		if commentIdx != -1 {
			lines[i] = line[:commentIdx]
		}
	}

	return strings.Join(lines, "\n")
}

func (c *DefaultClient) GetConnection() *sql.Conn {
	return c.conn
}

func (c *DefaultClient) Close() error {
	return c.conn.Close()
}

func IsArrayType(typeName string) bool {
	return len(typeName) >= 6 && typeName[:5] == "Array"
}

func GetBaseType(typeName string) string {
	if IsArrayType(typeName) {
		return typeName[6 : len(typeName)-1]
	}
	return typeName
}
