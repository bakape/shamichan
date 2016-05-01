// Core websocket message handlers

import {message, handlers} from './connection'

type SyncResonse = {

}

// Syncronise to the server and start receiving updates on the apropriate
// channel
handlers[message.synchronise] = (msg: SyncResonse) => {

}
