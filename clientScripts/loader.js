/*
 Selects and loads the client files
 Use only pure ES5.
*/

(function () {
	if ('serviceWorker' in navigator) {
		navigator.serviceWorker.register("/worker.js")
	} else {
		alert("Browser outdated. Install latest Chrome/Firefox/Opera")
		return
	}

	this.initModuleLoader(function (legacy) {
		return System.import("vendor/dom4").then(function () {
			// Application entry point
			return System.import("es" + (legacy ? 5 : 6) + '/client/main')
		})
	})
})()
