/*
 ServiceWorker entry point
*/

self.onmessage = msg =>
    console.log(msg.data3)

// TODO: Add selective caching logic
self.addEventListener("fetch", event =>
    event.respondWith(fetch(event.request)))
