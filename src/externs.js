'use strict'

mergeInto(LibraryManager.library, {
	local_storage_set: function (key, val) {
		localStorage.setItem(Pointer_stringify(key), Pointer_stringify(val))
	},
	local_storage_get: function (key) {
		var s = localStorage.getItem(key) || ""
		var len = s.length + 1
		var buf = Module._malloc(len)
		stringToUTF8(s, buf, len)
		return buf
	},
	page_path: function () {
		var s = location.pathname
		var len = s.length + 1
		var buf = Module._malloc(len)
		stringToUTF8(s, buf, len)
		return buf
	},
	page_query: function () {
		var s = location.search
		var len = s.length + 1
		var buf = Module._malloc(len)
		stringToUTF8(s, buf, len)
		return buf
	},
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
	},
	get_config: function () {
		var s = JSON.stringify(window.config)
		var len = s.length + 1
		var buf = Module._malloc(len)
		stringToUTF8(s, buf, len)
		return buf
	}
})
