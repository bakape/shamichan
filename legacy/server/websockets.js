/*
 Websocket handler module
 */

const _ = require('underscore'),
	caps = require('./caps'),
    common = require('../common/index'),
    events = require('events'),
	hooks = require('../util/hooks'),
    Muggle = require('../util/etc').Muggle,
    state = require('./state'),
    util = require('util'),
    winston = require('winston')

export const dispatcher = {}

/**
 * Websocket message parser and connection handler
 */
export class Client extends events.EventEmitter {
	/**
	 * Create websocket client
	 * @param {net.Socket} socket
	 * @param {string} ip
	 */
	constructor(socket, ip) {
		super()
		this.socket = socket
		this.ident = caps.lookUpIdent(ip)
		state.clients.add(this)
		state.countIPs()
	}

	/**
	 * Strigify and send message through websocket
	 * @param {Array} msg
	 */
	send(msg) {
		this.socket.write(JSON.stringify([msg]))
	}

	//TODO: Logic for redirecting from dead threads

	/**
	 * Parse incomming websocket message and dispatch to appropriate handler
	 * @param {string} data
	 */
	onMessage(data) {
		if (this.ident.ban)
			return
		let msg
		try {
			msg = JSON.parse(msg)
		}
		catch (e) {
			winston.warn('Unparsable websocket message:', data)
		}
		let type
		if (msg) {
			// Less overhead for sending post text body fragmets.
			if (this.post && typeof msg == 'string')
				type = common.UPDATE_POST
			else if (msg instanceof Array)
				type = msg.shift()
		}
		if (!this.synced && type !== common.SYNCHRONIZE)
			type = null
		const func = dispatcher[type]
		if (!func || !func(msg, this)) {
			this.disconnect(Muggle('Bad protocol:',
				new Error('Invalid message: ' + JSON.stringify(data))))
		}
	}

	/**
	 * Close all client listers on socket close
	 */
	onClose() {
		const {ip} = this.ident
		state.clients.delete(this)
		this.synced = false
		if (this.db)
			this.db.disconnect()
		this.emit('close')
	}

	/**
	 * Forcefully disconnect client due to error
	 * @param {(Muggle|Error)} error
	 */
	disconnect(error) {
		let msg = 'Server error'
		if (error instanceof Muggle) {
			msg = error.most_precise_error_message()
			error = error.deepest_reason()
		}
		winston.error(`Error by ${JSON.stringify(this.ident)}: ${error || msg}`)
		this.send([common.INVALID, msg])
		this.synced = false
	}
}

/**
 * Rescan connected client permissions and disconnect, if needed
 */
function scan_client_caps() {
	for (let client of state.clients) {
		if (!caps.lookUpIdent(client.ident.ip).ban)
			continue
		client.ident.ban = true
		client.disconnect(Muggle('Banned'))
	}
}
exports.scan_client_caps = scan_client_caps

/**
 * Push a message to all connected websocket clients
 * @param {*} msg
 */
export function push(msg){
	for (let client of _.values(state.clients)) {
		try {
			client.send(msg)
		}
		catch(e){
			// Client died, but we don't care
		}
	}
}

/**
 * Returns online count message
 * @param {int} count
 * @returns {*[]}
 */
function countMessage(count) {
	return [0, common.ONLINE_COUNT, count]
}

/**
 * Push online count on client connection
 */
hooks.hook('clientSynced', (info, cb) => {
	info.client.send(countMessage(state.IPCount))
	cb(null)
})

/**
 * Push online count update to all clients on change
 */
state.emitter.on('change:clientCount', count => push(countMessage(count)))
