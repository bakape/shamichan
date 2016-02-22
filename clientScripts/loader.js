/*
 Selects and loads the client files
 Use only pure ES5.
*/

(function () {
	if ('serviceWorker' in navigator) {
		navigator.serviceWorker
			.register("/ass/js/scripts/workerLoader.js")
			.then(function() {
				return navigator.serviceWorker.ready
			})
	} else {
		alert("Install Gentoo")
		return
	}

	this.initModuleLoader(function (legacy) {
		// Application entry point
		return System.import("vendor/dom4").then(function () {
			return System.import("es" + (legacy ? 5 : 6) + '/client/main')
		})
	})
})()
