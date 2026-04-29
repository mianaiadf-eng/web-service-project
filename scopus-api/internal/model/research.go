package model  

type Research struct {
	Title      string
	Journal    string
	Year       int
	DOI        *string
	Cited      int
	University string
} 

type User struct {
	UserID   string
	Password string
	APIKey   string
	Package  string
}