/*
 Stores the state of the ServiceWorker
*/

import {fetchJSON} from '../common'

export let config: {[key: string]: any}

export async function fetchConfig() {
    config = await fetchJSON('config')
}
