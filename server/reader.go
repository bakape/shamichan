package server

// Reader reads on formats thread and post structs
type Reader struct {
	board                             string
	ident                             Ident
	canSeeMnemonics, canSeeModeration bool
}

// NewReader constructs a new Reader struct
func NewReader(board string, ident Ident) *Reader {
	return &Reader{
		board,
		ident,

		// DJs  can see mnemonics, but not commited moderation
		ident.Auth == "dj" || CheckAuth("moderator", ident),
		CheckAuth("janitor", ident),
	}
}

// CheckAuth checks if the suplied Ident has enough or greater access right
// than requiered
func CheckAuth(auth string, ident Ident) bool {
	return authRank(auth) <= authRank(ident.Auth)
}

// authRank determines the rank of the suplied authority class in the access
// level hierarchy
func authRank(auth string) int {
	for i, level := range [4]string{"dj", "janitor", "moderator", "admin"} {
		if auth == level {
			return i
		}
	}
	return -1
}
