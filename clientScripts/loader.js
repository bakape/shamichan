/*
 Selects and loads the client files
 Use only pure ES5.
*/

(function () {
	if (!navigator.serviceWorker || typeof Promise === 'undefined') {
		alert("Browser outdated. Install latest Chrome/Firefox/Opera")
		return
	}
	var legacy
	navigator.serviceWorker.register("/worker.js").then(function () {
		return initModuleLoader()
	}).then(function (leg) {
		legacy = leg
		return System.import("vendor/dom4")
	}).then(function () {
		// Wait until serviceWorker is ready
		return navigator.serviceWorker.ready
	}).then(function () {
		return System.import("es" + (legacy ? 5 : 6) + '/client/main')
	})
})()
