'use strict'

mergeInto(LibraryManager.library, {
	set_outer_html: function (id, html) {
		document.getElementById(Pointer_stringify(id))
			.outerHTML = Pointer_stringify(html)
		return
	},
	set_inner_html: function (id, html) {
		document.getElementById(Pointer_stringify(id))
			.innerHTML = Pointer_stringify(html)
		return
	},
	get_inner_html: function (id) {
		var el = document.getElementById(Pointer_stringify(id))
		var s = el ? el.innerHTML : ""
		var len = s.length + 1
		var buf = Module._malloc(len)
		stringToUTF8(s, buf, len)
		return buf
	},
	append: function (id, html) {
		var cont = document.createElement('template')
		cont.innerHTML = Pointer_stringify(html)
		document.getElementById(Pointer_stringify(id))
			.appendChild(cont.content.firstChild)
	},
	prepend: function (id, html) {
		var cont = document.createElement('template')
		cont.innerHTML = Pointer_stringify(html)
		var el = document.getElementById(Pointer_stringify(id))
		el.insertBefore(cont.content.firstChild, el.firstChild)
	},
	before: function (id, html) {
		var cont = document.createElement('template')
		cont.innerHTML = Pointer_stringify(html)
		var el = document.getElementById(Pointer_stringify(id))
		el.parentNode.insertBefore(cont.content.firstChild, el)
	},
	after: function (id, html) {
		var cont = document.createElement('template')
		cont.innerHTML = Pointer_stringify(html)
		var el = document.getElementById(Pointer_stringify(id))
		el.parentNode.insertBefore(cont.content.firstChild, el.nextSibling)
	},
	local_storage_set: function (key, val) {
		localStorage.setItem(key, val)
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
	}
})
