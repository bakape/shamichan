/*
 Contains various general helper functions
*/

package main

// throw panics, if there is an error. Rob Pike must never know.
func throw(err error) {
	if err != nil {
		panic(err)
	}
}

// checkAuth checks if the suplied Ident has enough or greater access right
// than requiered
func checkAuth(auth string, ident Ident) bool {
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
