/**
 * Entry point of image processor
 */

const config = require('../config'),
    db = require('./db'),
    path = require('path'),
    redisDB = require('redis'),
    winston = require('winston')

exports.ClientController = db.ClientController

export const image_attrs = ('src thumb ext dims size MD5 SHA1 hash imgnm'
	+ 'spoiler apng mid audio length').split(' ')

/**
 * Creates redis client. Buffers commands, so no need for callback.
 * @returns {redis}
 */
export function redisClient() {
	const client = redisDB.createClient({
		host: config.redis_host,
		port: config.REDIS_PORT
	})
	client.select(config.redis_database || 0)
	client.on('error', err =>
		winston.error('Redis error:', err))
	return client
}
const redis = global.redis = redisClient()

export function media_path(dir, filename) {
	return path.join(config.MEDIA_DIRS[dir], filename)
}

export function squish_MD5 (hash) {
	if (typeof hash == 'string')
		hash = new Buffer(hash, 'hex');
	return hash.toString('base64').replace(/\//g, '_').replace(/=*$/, '')
}

// Start imager daemon
require('./daemon')
