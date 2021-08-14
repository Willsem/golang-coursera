package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
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
	idName  string
	columns []column
}

type DbExplorer struct {
	db     *sql.DB
	tables map[string]table
}

func NewDbExplorer(db *sql.DB) (DbExplorer, error) {
	rows, err := db.Query("show tables")
	if err != nil {
		return DbExplorer{}, err
	}

	defer rows.Close()

	tables := make(map[string]table, 0)

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

			if key == "PRI" {
				table.idName = column.name
			}

			table.columns = append(table.columns, column)
		}

		tables[table.name] = table

		col.Close()
	}

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
	tables := make([]string, 0, len(explorer.tables))
	for k := range explorer.tables {
		tables = append(tables, k)
	}

	result, err := json.Marshal(tables)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(result)
}

// GET /$table?limit=5&offset=7
func (explorer DbExplorer) getRowsFromTable(w http.ResponseWriter, r *http.Request) {
	table := r.URL.Path[1:]

	limit, err := getIntQueryParam(r, "limit", DEFAULT_LIMIT)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	offset, err := getIntQueryParam(r, "offset", DEFAULT_OFFSET)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rows, err := explorer.db.Query("select * from "+table+" limit ? offset ?", limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	explorer.sendRowsData(w, table, rows)
}

// GET /$table/$id
func (explorer DbExplorer) getRowFromTable(w http.ResponseWriter, r *http.Request) {
	params := strings.Split(r.URL.Path, "/")
	table := params[1]
	id := params[2]

	rows, err := explorer.db.Query("select * from "+table+" where "+explorer.tables[table].idName+" = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	explorer.sendRowsData(w, table, rows)
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

func getIntQueryParam(r *http.Request, param string, defaultValue int) (result int, err error) {
	value := r.URL.Query().Get(param)

	if value == "" {
		result = defaultValue
	} else {
		result, err = strconv.Atoi(value)
	}

	return
}

func (explorer DbExplorer) sendRowsData(w http.ResponseWriter, table string, rows *sql.Rows) {
	countColumns := len(explorer.tables[table].columns)
	row := make([]interface{}, countColumns)
	rowPtr := make([]interface{}, countColumns)
	for i := range row {
		rowPtr[i] = &row[i]
	}

	result := make([]map[string]interface{}, 0)

	for rows.Next() {
		rows.Scan(rowPtr...)
		result = append(result, explorer.readColumns(table, row))
	}

	if len(result) == 0 {
		http.Error(w, "not found in database", http.StatusNotFound)
		return
	}

	data, err := json.Marshal(result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(data)
}

func (explorer DbExplorer) readColumns(table string, row []interface{}) map[string]interface{} {
	cols := make(map[string]interface{}, 0)

	for i := range row {
		datatype := explorer.tables[table].columns[i].datatype
		colname := explorer.tables[table].columns[i].name
		if row[i] != nil {
			if datatype == "int" {
				cols[colname] = row[i].(int64)
			} else if datatype == "float" {
				cols[colname] = row[i].(float64)
			} else if datatype == "string" {
				cols[colname] = string(row[i].([]uint8))
			}
		}

	}

	return cols
}
