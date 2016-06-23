package types

// User contains ID, password hash and board-related data of a registered user
// account
type User struct {
	ID        string     `gorethink:"id"`
	Password  []byte     `gorethink:"password"`
	Positions []Position `gorethink:"positions"`
}

// Position defines a position of authority on a certain board
type Position struct {
	Board    string `gorethink:"board"`
	Position string `gorethink:"position"`
}
