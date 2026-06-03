package testmodel

// generate:reset
type User1 struct {
	ID      int
	Name    string
	Tags    []string
	Meta    map[string]string
	Counter *int
}
