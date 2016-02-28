/*
 ServiceWorker entry point
*/

import {fetchConfig} from './state'

// TODO: Properly import from handler modules
import * as connection from './connection'

// TODO: Add selective caching logic
self.onfetch = event =>
    event.respondWith(fetch(event.request))

fetchConfig()
