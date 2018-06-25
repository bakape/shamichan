importScripts("/assets/js/vendor/almond.js")
importScripts("/assets/js/worker.js")

var handler;

self.addEventListener('install', function (e) {
	e.waitUntil(self.skipWaiting()); // Activate worker immediately
});

self.addEventListener('activate', function (e) {
	e.waitUntil(
		self.clients.claim() // Claim all existing tabs
			.then(function () {
				return fetch("/json/config");
			})
			.then(function (res) {
				return res.json();
			})
			.then(function (res) {
				self.configs = res;
				return fetch("/assets/lang/"
					+ configs.defaultLang
					+ "/common.json");
			})
			.then(function (res) {
				return res.json();
			})
			.then(function (res) {
				self.lang = res;
			})
	);
	var module = require("worker/main");
	module.start().then(function () {
		handler = module.onMessage;
	});
});

// Event handler of 'message' event must be added on the initial evaluation of
// worker script
self.addEventListener('message', function (e) {
	if (handler) {
		handler(e);
	}
});

self.addEventListener('notificationclick', function (e) {
	var data = e.notification.data;
	if (parseInt(data)) {
		self.clients.openWindow("/all/" + data)
	}
});
