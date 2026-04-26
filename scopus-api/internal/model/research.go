package model   // ✅ ต้องชื่อ model

type Research struct {
	Title      string
	Journal    string
	Year       int
	DOI        *string
	Cited      int
	Authors    []string
	University string
}