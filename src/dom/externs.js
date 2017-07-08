'use strict'

mergeInto(LibraryManager.library, {
	alert: function (msg) {
		alert(Pointer_stringify(msg))
	},
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
	append: function (id, html) {
		var cont = document.createElement('template')
		cont.innerHTML = Pointer_stringify(html)
		document.getElementById(Pointer_stringify(id))
			.appendChild(cont.content.firstChild)
	},
	append_by_selector: function (sel, html) {
		var cont = document.createElement('template')
		cont.innerHTML = Pointer_stringify(html)
		document.querySelector(Pointer_stringify(sel))
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
})
