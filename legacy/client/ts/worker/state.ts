/*
 Stores the state of the ServiceWorker
*/

import {fetchJSON} from '../common'
import {open} from './db'

// Prepare worker for operation. Clients can not be handled, until this promise
// is resolved.
export const isReady = Promise.all([
    fetchConfig(),
    open()
])

export let config: {[key: string]: any}

// Fetch configuration from server
async function fetchConfig() {
    config = await fetchJSON('config')
}
