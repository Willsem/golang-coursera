package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

type Content struct {
	XMLName xml.Name `xml:"root"`
	Users   []Row    `xml:"row"`
}

type Row struct {
	Id     int    `xml:"id"`
	Name   string `xml:"first_name"`
	Age    int    `xml:"age"`
	About  string `xml:"about"`
	Gender string `xml:"gender"`
}

const (
	AccessToken = "authorized token"
	dataFile    = "dataset.xml"
)

var (
	ts         = httptest.NewServer(http.HandlerFunc(SearchServer))
	defaultReq = SearchRequest{
		Limit:  1,
		Offset: 1,
	}
	content Content
)

func init() {
	file, err := os.Open(dataFile)
	if err != nil {
		panic(err)
	}

	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	err = xml.Unmarshal([]byte(fileContents), &content)
	if err != nil {
		panic(err)
	}
}

func (r Row) convert() User {
	return User{
		Id:     r.Id,
		Name:   r.Name,
		Age:    r.Age,
		About:  r.About,
		Gender: r.Gender,
	}
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}()

	accsessToken := r.Header.Get("AccessToken")
	if accsessToken != AccessToken {
		http.Error(w, "401 - unauthorized", http.StatusUnauthorized)
		return
	}

	limit, err := strconv.Atoi(r.FormValue("limit"))
	if err != nil {
		http.Error(w, "400 - bad query", http.StatusBadRequest)
		return
	}

	offset, err := strconv.Atoi(r.FormValue("offset"))
	if err != nil {
		http.Error(w, "400 - bad query", http.StatusBadRequest)
		return
	}

	var users []string
	if limit > 25 {
		limit = 25
	}
	if offset+limit > len(content.Users) {
		limit = len(content.Users)
	}
	for i := offset; i < limit; i++ {
		user := content.Users[i].convert()
		u, err := json.Marshal(user)
		if err != nil {
			panic(err)
		}
		users = append(users, string(u))
	}

	w.WriteHeader(http.StatusOK)
	io.WriteString(w, `[`+strings.Join(users, ",")+`]`)
}

func TestFindUsersOk(t *testing.T) {
	srv := SearchClient{
		URL:         ts.URL,
		AccessToken: AccessToken,
	}

	cases := []SearchRequest{
		{
			Limit:  1,
			Offset: 0,
		},
		{
			Limit:  30,
			Offset: 26,
		},
	}

	for _, req := range cases {
		_, err := srv.FindUsers(req)
		if err != nil {
			t.Error("unexpected error:", err)
		}
	}
}

func TestFindUsersErrorWithRequest(t *testing.T) {
	srv := SearchClient{}

	cases := []SearchRequest{
		{
			Limit: -1,
		},
		{
			Limit:  0,
			Offset: -1,
		},
		{
			Limit:  26,
			Offset: -1,
		},
	}

	for _, req := range cases {
		_, err := srv.FindUsers(req)

		if err == nil {
			t.Error("expected error")
		}
	}
}

func TestFindUsersUnauthorized(t *testing.T) {
	srv := SearchClient{
		URL:         ts.URL,
		AccessToken: "bad token",
	}

	_, err := srv.FindUsers(defaultReq)
	if err == nil {
		t.Error("expected error")
	}
	if err.Error() != "Bad AccessToken" {
		t.Error("incorrect error:\n\tgot:", err.Error(), "\n\texpected: Bad AccessToken")
	}
}

func emptyServer(w http.ResponseWriter, r *http.Request) {
	return
}

func TestFindUsersUnmarshalError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(emptyServer))
	errorTest(t, ts.URL)
}

func TestFindUsersNilServer(t *testing.T) {
	errorTest(t, "")
}

func timeoutServer(w http.ResponseWriter, r *http.Request) {
	time.Sleep(1 * time.Second)
}

func TestFindUsersServerTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(timeoutServer))
	errorTest(t, ts.URL)
}

func internalServer(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Internal error", http.StatusInternalServerError)
}

func TestFindUsersInternalError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(internalServer))
	errorTest(t, ts.URL)
}

func badRequestServer(w http.ResponseWriter, r *http.Request) {
	limit, err := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	if err != nil {
		http.Error(w, "Cannot convert limit param", http.StatusBadRequest)
	}

	switch limit {
	case 1:
		http.Error(w, "undefined error", http.StatusBadRequest)
	case 2:
		http.Error(w, `{"Error":"ErrorBadOrderField"}`, http.StatusBadRequest)
	default:
		http.Error(w, `{"Error":"undefined"}`, http.StatusBadRequest)
	}
}

func TestFindUsersBadRequestError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(badRequestServer))
	srv := SearchClient{
		URL: ts.URL,
	}

	cases := []SearchRequest{
		{
			Limit: 0,
		},
		{
			Limit: 1,
		},
		{
			Limit: 2,
		},
	}

	for _, req := range cases {
		_, err := srv.FindUsers(req)
		if err == nil {
			t.Error("expected error")
		}
	}
}

func errorTest(t *testing.T, URL string) {
	srv := SearchClient{
		URL: URL,
	}

	_, err := srv.FindUsers(defaultReq)
	if err == nil {
		t.Error("expected error")
	}
}
