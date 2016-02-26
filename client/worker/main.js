/*
 ServiceWorker entry point
*/

import {fetchConfig} from './state'
import * as clients from './clients'

// TODO: Add selective caching logic
self.onfetch = event =>
    event.respondWith(fetch(event.request))
