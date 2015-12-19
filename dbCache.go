/*
 Functions that operate on the post thread and board parenthood cache
*/

package main

import (
	r "github.com/dancannon/gorethink"
	"strconv"
)

type updateMap map[string]r.Term

// cacheAdd adds a post to the parenthood cache and increments board history
// counters
func cacheAdd(id int, op int, board string) {
	num := strconv.Itoa(id)
	rExec(r.Table("main").Get("cache").Update(ParenthoodCache{
		intMap{num: op},
		stringMap{num: board},
	}))
}

// cacheRemove removes a post from the parenthood cache
func cacheRemove(id int) {
	num := strconv.Itoa(id)
	rExec(r.Table("main").Get("cache").Replace(r.Row.Without(updateMap{
		"OPs":    removeField(num),
		"boards": removeField(num),
	})))
}

func removeField(num string) r.Term {
	return r.Row.Without(num)
}

// parentThread determines the parent thread of a post
func parentThread(id int) (op int) {
	query := r.Table("main").Get("cache").
		Field("OPs").
		Field(strconv.Itoa(id))
	rGet(query).One(&op)
	return
}

// parentBoard determines the parent board of the post
func parentBoard(id int) (board string) {
	query := r.Table("main").Get("cache").
		Field("boards").
		Field(strconv.Itoa(id))
	rGet(query).One(&board)
	return
}

// ValidateOP confirms the specified thread exists on specific board
func validateOP(id int, board string) bool {
	return parentBoard(id) == board && parentThread(id) == id
}
