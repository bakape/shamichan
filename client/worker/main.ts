/*
 ServiceWorker entry point
*/

import {connect} from './connection'

// TODO: Add selective caching logic
self.onfetch = event =>
    event.respondWith(fetch(event.request))

connect()
