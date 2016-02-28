/*
 ServiceWorker entry point
*/

import {connect} from './connection'
import {fetchConfig} from './state'

// TODO: Add selective caching logic
self.onfetch = event =>
    event.respondWith(fetch(event.request))

self.onactivate = event =>
    event.waitUntil(Promise.all([
        connect(),
        self.clients.claim(),
        fetchConfig()
    ]))
