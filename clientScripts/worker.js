/*
 Loads the service worker using SystemJS
*/

importScripts("/ass/js/vendor/system.js", "/ass/js/scripts/loaderCommon.js")

self.oninstall = function (event) {
    event.waitUntil(self.initModuleLoader().then(function (legacy) {
        return System.import("es" + (legacy ? 5 : 6) + "/worker/main")
    }))
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
