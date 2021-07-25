package user

//easyjson:json
type User struct {
	Browsers []string `json:"browsers"`
	Email    string   `json:"email"`
	Name     string   `json:"name"`
	company  string
	country  string
	job      string
	phone    string
}
