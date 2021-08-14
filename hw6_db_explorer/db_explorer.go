package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

const (
	DEFAULT_LIMIT  = 5
	DEFAULT_OFFSET = 0
)

var (
	tableRegexp       = regexp.MustCompile(`^\/[^\/]+\/?$`)
	tableWithIdRegexp = regexp.MustCompile(`^\/[^\/]+\/.+`)
)

type column struct {
	name     string
	datatype string
	nullable bool
}

type table struct {
	name    string
	columns []column
}

type DbExplorer struct {
	db     *sql.DB
	tables []table
}

func NewDbExplorer(db *sql.DB) (DbExplorer, error) {
	rows, err := db.Query("show tables")
	if err != nil {
		return DbExplorer{}, err
	}

	defer rows.Close()

	tables := make([]table, 0)

	for rows.Next() {
		var nameTable string
		err = rows.Scan(&nameTable)
		if err != nil {
			return DbExplorer{}, err
		}

		table := table{
			name:    nameTable,
			columns: make([]column, 0),
		}

		col, err := db.Query("show columns from " + table.name)
		if err != nil {
			return DbExplorer{}, err
		}

		for col.Next() {
			column := column{}
			var nullable string
			var key string
			var def string
			var ext string

			col.Scan(&column.name, &column.datatype, &nullable, &key, &def, &ext)
			if nullable == "YES" {
				column.nullable = true
			} else {
				column.nullable = false
			}

			if strings.Contains(column.datatype, "varchar") || column.datatype == "text" {
				column.datatype = "string"
			}

			table.columns = append(table.columns, column)
		}

		col.Close()
	}

	fmt.Println(tables)

	return DbExplorer{
		db:     db,
		tables: tables,
	}, nil
}

func (explorer DbExplorer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET" && r.URL.Path == "/":
		explorer.getTables(w, r)
	case r.Method == "GET" && tableRegexp.MatchString(r.URL.Path):
		explorer.getRowsFromTable(w, r)
	case r.Method == "GET" && tableWithIdRegexp.MatchString(r.URL.Path):
		explorer.getRowFromTable(w, r)
	case r.Method == "PUT" && tableRegexp.MatchString(r.URL.Path):
		explorer.putRowToTable(w, r)
	case r.Method == "POST" && tableWithIdRegexp.MatchString(r.URL.Path):
		explorer.postRowInTable(w, r)
	case r.Method == "DELETE" && tableWithIdRegexp.MatchString(r.URL.Path):
		explorer.deleteRowFromTable(w, r)
	default:
		http.Error(w, "unknown method", http.StatusNotFound)
	}
}

// GET /
func (explorer DbExplorer) getTables(w http.ResponseWriter, r *http.Request) {
	result, err := json.Marshal(explorer.tables)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(result)
}

// GET /$table?limit=5&offset=7
func (explorer DbExplorer) getRowsFromTable(w http.ResponseWriter, r *http.Request) {
}

// GET /$table/$id
func (explorer DbExplorer) getRowFromTable(w http.ResponseWriter, r *http.Request) {
}

// PUT /$table
func (explorer DbExplorer) putRowToTable(w http.ResponseWriter, r *http.Request) {
}

// POST /$table/$id
func (explorer DbExplorer) postRowInTable(w http.ResponseWriter, r *http.Request) {
}

// DELETE /$table/$id
func (explorer DbExplorer) deleteRowFromTable(w http.ResponseWriter, r *http.Request) {
}
