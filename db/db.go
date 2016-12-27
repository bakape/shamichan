// Package db handles all core database interactions of the server
package db

// var postReservationQuery = GetMain("info").
// 	Update(
// 		map[string]r.Term{
// 			"postCtr": r.Row.Field("postCtr").Add(1),
// 		},
// 		r.UpdateOpts{
// 			ReturnChanges: true,
// 		},
// 	).
// 	Field("changes").
// 	AtIndex(0).
// 	Field("new_val").
// 	Field("postCtr")

// // FindPost finds a post only by ID number
// func FindPost(id uint64) r.Term {
// 	return r.Table("posts").Get(id)
// }

// // ValidateOP confirms the specified thread exists on specific board
// func ValidateOP(id uint64, board string) (valid bool, err error) {
// 	err = One(FindThread(id).Field("board").Eq(board).Default(false), &valid)
// 	return
// }

// // FindThread is a  shorthand for retrieving a document from the "threads" table
// func FindThread(id uint64) r.Term {
// 	return r.Table("threads").Get(id)
// }

// // GetMain is a shorthand for retrieving a document from the "main" table
// func GetMain(id string) r.Term {
// 	return r.Table("main").Get(id)
// }

// // GetAccount is a shorthand for retrieving a document from the "accounts" table
// func GetAccount(id string) r.Term {
// 	return r.Table("accounts").Get(id)
// }

// // GetImage is a shorthand for retrieving a document from the "images" table
// func GetImage(id string) r.Term {
// 	return r.Table("images").Get(id)
// }

// // Insert is a shorthand for inserting documents or slices of documents into a
// // table
// func Insert(table string, doc interface{}) error {
// 	return Write(r.Table(table).Insert(doc))
// }

// // BoardCounter retrieves the history or "progress" counter of a board
// func BoardCounter(board string) (counter uint64, err error) {
// 	q := r.
// 		Table("posts").
// 		GetAllByIndex("board", board).
// 		Field("lastUpdated").
// 		Max().
// 		Default(0)
// 	err = One(q, &counter)
// 	return
// }

// // ThreadCounter retrieves the post counter of a thread to get a rough estimate
// // of the thread's progress
// func ThreadCounter(id uint64) (counter uint64, err error) {
// 	q := r.
// 		Table("posts").
// 		GetAllByIndex("op", id).
// 		Field("lastUpdated").
// 		Max().
// 		Default(0)
// 	err = One(q, &counter)
// 	return
// }

// // GetLoginHash retrieves the login hash of the registered user account
// func GetLoginHash(id string) (hash []byte, err error) {
// 	query := GetAccount(id).Field("password").Default(nil)
// 	err = One(query, &hash)
// 	return
// }

// // ReservePostID reserves a post ID number for post and thread creation
// func ReservePostID() (id uint64, err error) {
// 	err = One(postReservationQuery, &id)
// 	return
// }
