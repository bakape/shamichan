/*
 Loads the service worker using SystemJS
*/

importScripts("/ass/js/vendor/system.js", "/ass/js/scripts/loaderCommon.js")
initModuleLoader()

self.oninstall = function (event) {
    event.waitUntil(System.import("worker/main"))
}

self.onactivate = function (event) {
    event.waitUntil(Promise.all([
        fetchConfig(),
        self.clients.claim()
    ]))
}

function fetchConfig() {
    return fetch("/api/config").then(function (res) {
        return res.json()
    }).then(function (json) {
        self.config = json
    })
}
