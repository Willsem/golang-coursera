package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

type Response struct {
	Response map[string]interface{} `json:"response"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

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

	tables := make(map[string]table)

	for rows.Next() {
		var nameTable string
		err = rows.Scan(&nameTable)
		if err != nil {
			return DbExplorer{}, err
		}

		tables[nameTable] = table{
			name:    nameTable,
			columns: make([]column, 0),
		}
	}

	rows.Close()

	for nameTable := range tables {
		col, err := db.Query("show columns from " + nameTable)
		if err != nil {
			return DbExplorer{}, err
		}

		table := tables[nameTable]

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

		tables[nameTable] = table

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

	response := Response{
		Response: map[string]interface{}{
			"tables": tables,
		},
	}

	result, _ := json.Marshal(response)
	w.Write(result)
}

// GET /$table?limit=5&offset=7
func (explorer DbExplorer) getRowsFromTable(w http.ResponseWriter, r *http.Request) {
	table := strings.Split(r.URL.Path, "/")[1]

	limit := getIntQueryParam(r, "limit", DEFAULT_LIMIT)
	offset := getIntQueryParam(r, "offset", DEFAULT_OFFSET)

	rows, err := explorer.db.Query("select * from "+table+" limit ? offset ?", limit, offset)
	if err != nil {
		errorMessage, _ := json.Marshal(ErrorResponse{Error: "unknown table"})
		http.Error(w, string(errorMessage), http.StatusNotFound)
		return
	}
	defer rows.Close()

	result := explorer.getRowsData(table, rows, false)
	if result == nil {
		errorMessage, _ := json.Marshal(ErrorResponse{Error: "records not found"})
		http.Error(w, string(errorMessage), http.StatusNotFound)
		return
	}

	response := Response{
		Response: map[string]interface{}{
			"records": result,
		},
	}

	data, _ := json.Marshal(response)
	w.Write(data)
}

// GET /$table/$id
func (explorer DbExplorer) getRowFromTable(w http.ResponseWriter, r *http.Request) {
	params := strings.Split(r.URL.Path, "/")
	table := params[1]
	id := params[2]

	rows, err := explorer.db.Query("select * from "+table+" where "+explorer.tables[table].idName+" = ?", id)
	if err != nil {
		http.Error(w, "null", http.StatusNotFound)
		return
	}
	defer rows.Close()

	result := explorer.getRowsData(table, rows, true)
	if result == nil {
		errorMessage, _ := json.Marshal(ErrorResponse{Error: "record not found"})
		http.Error(w, string(errorMessage), http.StatusNotFound)
		return
	}

	response := Response{
		Response: map[string]interface{}{
			"record": result,
		},
	}

	data, _ := json.Marshal(response)
	w.Write(data)
}

// PUT /$table
func (explorer DbExplorer) putRowToTable(w http.ResponseWriter, r *http.Request) {
	body, err := explorer.parseBody(r)
	if err != nil {
		errorMessage, _ := json.Marshal(ErrorResponse{Error: err.Error()})
		http.Error(w, string(errorMessage), http.StatusBadRequest)
		return
	}

	table := strings.Split(r.URL.Path, "/")[1]
	columns := ""
	values := make([]interface{}, 0)
	placeholders := ""
	for key, value := range body {
		columns += ", `" + key + "`"
		values = append(values, value)
		placeholders += "?, "
	}

	columns = columns[2:]
	placeholders = placeholders[:len(placeholders)-2]

	result, err := explorer.db.Exec(
		"insert into "+table+" ("+columns+") values ("+placeholders+")",
		values...,
	)
	if err != nil {
		errorMessage, _ := json.Marshal(ErrorResponse{Error: err.Error()})
		fmt.Println(string(errorMessage))
		http.Error(w, string(errorMessage), http.StatusInternalServerError)
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		errorMessage, _ := json.Marshal(ErrorResponse{Error: err.Error()})
		http.Error(w, string(errorMessage), http.StatusInternalServerError)
		return
	}

	response := Response{
		Response: map[string]interface{}{
			explorer.tables[table].idName: id,
		},
	}
	data, _ := json.Marshal(response)
	w.Write(data)
}

// POST /$table/$id
func (explorer DbExplorer) postRowInTable(w http.ResponseWriter, r *http.Request) {
	body, err := explorer.parseBody(r)
	if err != nil {
		errorMessage, _ := json.Marshal(ErrorResponse{Error: err.Error()})
		http.Error(w, string(errorMessage), http.StatusBadRequest)
		return
	}

	params := strings.Split(r.URL.Path, "/")
	table := params[1]
	id := params[2]
	keys := ""
	values := make([]interface{}, 0)
	for key, value := range body {
		keys += "`" + key + "` = ? ,"
		values = append(values, value)
	}

	keys = keys[:len(keys)-1]

	result, err := explorer.db.Exec(
		"update "+table+" set "+keys+"where "+explorer.tables[table].idName+" = "+id,
		values...,
	)
	if err != nil {
		errorMessage, _ := json.Marshal(ErrorResponse{Error: err.Error()})
		http.Error(w, string(errorMessage), http.StatusInternalServerError)
		return
	}

	updated, err := result.RowsAffected()
	if err != nil {
		errorMessage, _ := json.Marshal(ErrorResponse{Error: err.Error()})
		http.Error(w, string(errorMessage), http.StatusInternalServerError)
		return
	}

	response := Response{
		Response: map[string]interface{}{
			"updated": updated,
		},
	}
	data, _ := json.Marshal(response)
	w.Write(data)
}

// DELETE /$table/$id
func (explorer DbExplorer) deleteRowFromTable(w http.ResponseWriter, r *http.Request) {
	params := strings.Split(r.URL.Path, "/")
	table := params[1]
	id := params[2]

	result, err := explorer.db.Exec(
		"delete from "+table+" where "+explorer.tables[table].idName+" = ?",
		id,
	)
	if err != nil {
		errorMessage, _ := json.Marshal(ErrorResponse{Error: err.Error()})
		http.Error(w, string(errorMessage), http.StatusInternalServerError)
		return
	}

	affected, err := result.RowsAffected()
	if err != nil {
		errorMessage, _ := json.Marshal(ErrorResponse{Error: err.Error()})
		http.Error(w, string(errorMessage), http.StatusInternalServerError)
		return
	}

	response := Response{
		Response: map[string]interface{}{
			"deleted": affected,
		},
	}
	data, _ := json.Marshal(response)
	w.Write(data)
}

func getIntQueryParam(r *http.Request, param string, defaultValue int) int {
	value := r.URL.Query().Get(param)

	if value == "" {
		return defaultValue
	} else {
		result, err := strconv.Atoi(value)
		if err != nil {
			return defaultValue
		} else {
			return result
		}
	}
}

func (explorer DbExplorer) getRowsData(table string, rows *sql.Rows, onlyOne bool) interface{} {
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
		return nil
	}

	if onlyOne {
		return result[0]
	} else {
		return result
	}
}

func (explorer DbExplorer) readColumns(table string, row []interface{}) map[string]interface{} {
	cols := make(map[string]interface{})

	for i := range row {
		datatype := explorer.tables[table].columns[i].datatype
		colname := explorer.tables[table].columns[i].name
		if row[i] == nil {
			cols[colname] = nil
		} else {
			switch datatype {
			case "int":
				cols[colname] = row[i].(int64)
			case "float":
				cols[colname] = row[i].(float64)
			case "string":
				cols[colname] = string(row[i].([]uint8))
			}
		}
	}

	return cols
}

func (explorer DbExplorer) parseBody(r *http.Request) (map[string]interface{}, error) {
	body, err := getBodyFromRequest(r)
	if err != nil {
		return nil, err
	}

	table := strings.Split(r.URL.Path, "/")[1]
	fields := explorer.tables[table].columns
	parsedBody := make(map[string]interface{})

	if r.Method == "PUT" {
		delete(body, explorer.tables[table].idName)
	}

	for key, value := range body {
		for _, col := range fields {
			if key == col.name {
				var ok bool
				switch col.datatype {
				case "int":
					parsedBody[key], ok = value.(int64)
				case "float":
					parsedBody[key], ok = value.(float64)
				case "string":
					parsedBody[key], ok = value.(string)
				}

				if !ok {
					if value == nil && col.nullable {
						parsedBody[key] = nil
					} else {
						return nil, fmt.Errorf("field " + key + " have invalid type")
					}
				}
			}
		}
	}

	if r.Method == "POST" {
		if _, ok := parsedBody[explorer.tables[table].idName]; ok {
			return nil, fmt.Errorf("field " + explorer.tables[table].idName + " have invalid type")
		}
	}

	if r.Method == "PUT" {
		for _, col := range fields {
			was := false
			for keys := range parsedBody {
				if col.name == keys {
					was = true
					break
				}
			}

			if !was {
				if col.nullable {
					parsedBody[col.name] = nil
				} else {
					switch col.datatype {
					case "int":
						parsedBody[col.name] = 0
					case "float":
						parsedBody[col.name] = 0.0
					case "string":
						parsedBody[col.name] = ""
					}
				}
			}
		}
	}

	return parsedBody, nil
}

func getBodyFromRequest(r *http.Request) (map[string]interface{}, error) {
	b, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		return nil, err
	}

	var body map[string]interface{}
	err = json.Unmarshal(b, &body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
