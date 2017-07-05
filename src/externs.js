'use strict'

mergeInto(LibraryManager.library, {
	alert: function (msg) {
		alert(Pointer_stringify(msg))
	},
	set_outer_HTML: function (id, html) {
		document.getElementById(Pointer_stringify(id))
			.outerHTML = Pointer_stringify(html)
		return
	},
	set_inner_html: function (id, html) {
		document.getElementById(Pointer_stringify(id))
			.innerHTML = Pointer_stringify(html)
		return
	},
	pop_children: function (id, n) {
		var el = document.getElementByID(Pointer_stringify(id))
		for (var i = 0; i <= n; i++) {
			el.lastChild.remove()
		}
	},
	append_element: function (id, html) {
		var cont = document.createElement('template')
		cont.innerHTML = Pointer_stringify(html)
		document.getElementByID(Pointer_stringify(id))
			.appendChild(cont.content.firstChild)
	}
})
