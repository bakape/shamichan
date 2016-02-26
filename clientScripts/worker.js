/*
 Loads the service worker using SystemJS
*/

importScripts("/ass/js/vendor/system.js", "/ass/js/scripts/loaderCommon.js")

self.addEventListener("install", function (event) {
    event.waitUntil(self.initModuleLoader(function (legacy) {
        return System.import("es" + (legacy ? 5 : 6) + "/worker/main")
    }))
})

self.addEventListener("activate", function (event) {
    event.waitUntil(self.clients.claim())
})
