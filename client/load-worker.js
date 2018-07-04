importScripts("/assets/js/vendor/almond.js")
importScripts("/assets/js/worker.js")

self.addEventListener('install', function () { });
self.addEventListener('activate', function () {
	require("worker/main");
});
