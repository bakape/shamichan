'use strict'

mergeInto(LibraryManager.library, {
	load_db: function (threads, len) {
		// The original array will be freed on the Rust side after db.open()
		var ops = new Array(len)
		for (var i = 0; i < len; i++) {
			ops[i] = getValue(threads + i * 8, 'i64')
		}

		var db = require("db")
		var read = db.readIDs
		var store = Module.cwrap("set_store", null, ["number", "array"])
		db.open()
			.then(function () {
				return Promise.all([
					read("mine", threads).then(function (ids) {
						store(0, ids)
					}),
					read("seen", threads).then(function (ids) {
						store(1, ids)
					}),
					read("seenPost", threads).then(function (ids) {
						store(2, ids)
					}),
					read("hidden", threads).then(function (ids) {
						store(3, ids)
					})
				])
			})
			.then(Module.cwrap("render_page", null, []))
	}
})
