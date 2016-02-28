/*
 Loads the service worker using SystemJS
*/

importScripts("/ass/js/vendor/system.js", "/ass/js/scripts/loaderCommon.js")
initModuleLoader()

self.oninstall = function (event) {
    event.waitUntil(System.import("worker/main"))
}

self.onactivate = function (event) {
    event.waitUntil(self.clients.claim())
}
