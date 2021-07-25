package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"

	"github.com/mailru/easyjson/jlexer"
	"github.com/mailru/easyjson/jwriter"
)

type User struct {
	Browsers []string `json:"browsers"`
	Email    string   `json:"email"`
	Name     string   `json:"name"`
	company  string
	country  string
	job      string
	phone    string
}

func easyjson9e1087fdDecodeGithubComWillsemGolangCourseraHw3BenchGenerate(in *jlexer.Lexer, out *User) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "browsers":
			if in.IsNull() {
				in.Skip()
				out.Browsers = nil
			} else {
				in.Delim('[')
				if out.Browsers == nil {
					if !in.IsDelim(']') {
						out.Browsers = make([]string, 0, 4)
					} else {
						out.Browsers = []string{}
					}
				} else {
					out.Browsers = (out.Browsers)[:0]
				}
				for !in.IsDelim(']') {
					var v1 string
					v1 = string(in.String())
					out.Browsers = append(out.Browsers, v1)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "email":
			out.Email = string(in.String())
		case "name":
			out.Name = string(in.String())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson9e1087fdEncodeGithubComWillsemGolangCourseraHw3BenchGenerate(out *jwriter.Writer, in User) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"browsers\":"
		out.RawString(prefix[1:])
		if in.Browsers == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
			out.RawString("null")
		} else {
			out.RawByte('[')
			for v2, v3 := range in.Browsers {
				if v2 > 0 {
					out.RawByte(',')
				}
				out.String(string(v3))
			}
			out.RawByte(']')
		}
	}
	{
		const prefix string = ",\"email\":"
		out.RawString(prefix)
		out.String(string(in.Email))
	}
	{
		const prefix string = ",\"name\":"
		out.RawString(prefix)
		out.String(string(in.Name))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v User) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson9e1087fdEncodeGithubComWillsemGolangCourseraHw3BenchGenerate(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v User) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson9e1087fdEncodeGithubComWillsemGolangCourseraHw3BenchGenerate(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *User) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson9e1087fdDecodeGithubComWillsemGolangCourseraHw3BenchGenerate(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *User) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson9e1087fdDecodeGithubComWillsemGolangCourseraHw3BenchGenerate(l, v)
}

const (
	maxUsers = 1000
)

var (
	androidRegexp, _ = regexp.Compile("Android")
	msieRegexp, _    = regexp.Compile("MSIE")
	r, _             = regexp.Compile("@")
)

func FastSearch(out io.Writer) {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	defer file.Close()

	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	lines := bytes.Split(fileContents, []byte{byte('\n')})
	seenBrowsers := make(map[string]bool, maxUsers)

	fmt.Fprintln(out, "found users:")
	user := User{}
	for i, line := range lines {
		err := user.UnmarshalJSON(line)
		if err != nil {
			panic(err)
		}

		isAndroid := false
		isMSIE := false

		for _, browser := range user.Browsers {
			isMatch := false

			if ok := androidRegexp.MatchString(browser); ok {
				isAndroid = true
				isMatch = true
			}

			if ok := msieRegexp.MatchString(browser); ok {
				isMSIE = true
				isMatch = true
			}

			if isMatch {
				if _, seenBefore := seenBrowsers[browser]; !seenBefore {
					seenBrowsers[browser] = true
				}
			}
		}

		if isAndroid && isMSIE {
			email := r.ReplaceAllString(user.Email, " [at] ")
			fmt.Fprintln(out, fmt.Sprintf("[%d] %s <%s>", i, user.Name, email))
		}
	}

	fmt.Fprintln(out, "\nTotal unique browsers", len(seenBrowsers))
}
