/*
 Functions that operate on the post thread and board parenthood cache
*/

package main

import (
	r "github.com/dancannon/gorethink"
)

type termMap map[string]r.Term

// cacheAdd adds a post to the parenthood cache and increments board history
// counters
func cacheAdd(id, op uint64, board string) {
	num := idToString(id)
	rExec(r.Table("main").Get("cache").Update(ParenthoodCache{
		map[string]uint64{num: op},
		map[string]string{num: board},
	}))
}

// cacheRemove removes a post from the parenthood cache
func cacheRemove(id uint64) {
	num := idToString(id)
	rExec(r.Table("main").Get("cache").Replace(r.Row.Without(termMap{
		"OPs":    removeField(num),
		"boards": removeField(num),
	})))
}

func removeField(num string) r.Term {
	return r.Row.Without(num)
}

// parentThread determines the parent thread of a post
func parentThread(id uint64) (op uint64) {
	query := r.Table("main").Get("cache").
		Field("OPs").
		Field(idToString(id)).
		Default(0)
	rGet(query).One(&op)
	return
}

// parentBoard determines the parent board of the post
func parentBoard(id uint64) (board string) {
	query := r.Table("main").Get("cache").
		Field("boards").
		Field(idToString(id)).
		Default("")
	rGet(query).One(&board)
	return
}

// ValidateOP confirms the specified thread exists on specific board
func validateOP(id uint64, board string) bool {
	return parentBoard(id) == board && parentThread(id) == id
}
