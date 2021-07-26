package main

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

const (
	AccessToken = "authorized token"
)

var (
	ts         = httptest.NewServer(http.HandlerFunc(SearchServer))
	defaultReq = SearchRequest{
		Limit:  1,
		Offset: 1,
	}
)

func SearchServer(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if recover() != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}()

	accsessToken := r.Header.Get("AccessToken")
	if accsessToken != AccessToken {
		http.Error(w, "401 - unauthorized", http.StatusUnauthorized)
		return
	}

	_, err := r.GetBody()
	if err != nil {
		http.Error(w, "400 - bad body", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()
}

func TestFindUsersOk(t *testing.T) {
	//_ := ts.URL
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
