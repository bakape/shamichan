/*
 Loads the service worker using SystemJS
*/

this.global = {}

importScripts("/ass/js/vendor/system.js", "/ass/js/scripts/loaderCommon.js")

initModuleLoader(function (legacy) {
    return System.import("es" + (legacy ? 5 : 6) + "/worker/main")
})

// var self = this
// this.addEventListener("install", function (event) {
//     event.waitUntil(function (legacy) {
//         return
//     })
// })
