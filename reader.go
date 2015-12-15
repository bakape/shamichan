package main

// Reader reads on formats thread and post structs
type Reader struct {
	board                             string
	ident                             Ident
	canSeeMnemonics, canSeeModeration bool
}

// Newreader constructs a new Reader struct
func Newreader(board string, ident Ident) *Reader {
	return &Reader{
		board,
		ident,

		// DJs  can see mnemonics, but not commited moderation
		ident.Auth == "dj" || checkAuth("moderator", ident),
		checkAuth("janitor", ident),
	}
}
