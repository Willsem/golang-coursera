package main

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

var testOk = "1\n2\n3\n4\n5"
var testOkResult = "1\n2\n3\n4\n5\n"

func TestOk(t *testing.T) {
	in := bufio.NewReader(strings.NewReader(testOk))
	out := new(bytes.Buffer)
	err := uniq(in, out)

	if err != nil {
		t.Errorf("test OK Failed")
	}

	if result := out.String(); result != testOkResult {
		t.Errorf("test for OK Failed - results not match\n%v\n%v", result, testOkResult)
	}
}

var testFail = "1\n2\n1"

func TetForError(t *testing.T) {
	in := bufio.NewReader(strings.NewReader(testFail))
	out := new(bytes.Buffer)
	err := uniq(in, out)

	if err == nil {
		t.Errorf("test Error Failed - error")
	}
}
