const config = require('../config'),
    db = require('../db'),
    fs = require('fs-extra'),
    Promise = require('bluebird'),
    {redis} = global,
    state = require('./state'),
    validate = require('./validate_message'),
    winston = require('winston')

/**
 * Initialize imager functionality
 */
module.exports = async function init() {
    await makeMediaDirs()
    await setupImageRelay()
    if (!config.READ_ONLY)
        await deleteTemps()
}

/**
 * Create the directories requied for image processing
 */
async function makeMediaDirs() {
	const dirs = ['src', 'thumb', 'tmp']
	if (config.EXTRA_MID_THUMBNAILS)
		dirs.push('mid')
    await Promise.all(dirs.map(async dir => {
        await fs.mkdirAsync(config.MEDIA_DIRS[dir]).catch(err => {
            if (err.code !== 'EEXIST')
                throw error
        })
    }))
}

/**
 * Setup the imager -> server -> client communication pathway
 */
async function setupImageRelay() {
    const redis = db.redisClient()
    redis.psubscribe('client:*')

    // Convert event to promise
    await new Promise(resolve =>
        redis.once('psubscribe', resolve))

    // Send image status update to client, if any
    redis.on('pmessage', (pattern, chan, status) => {
        const clientID = parseInt(chan.match(/^client:(\d+)$/)[1], 10)
        if (!clientID || !validate.value('id', clientID))
            return
    	const client = state.clients[clientID]
    	if (client) {
    		try {
    			client.send([0, common.IMAGE_STATUS, status]);
    		}
    		catch (e) {
    			// Swallow EINTR
    			// anta baka?
    		}
    	}
    })
}

/**
 * Catch any dangling images on server startup
 */
async function deleteTemps() {
    const temps = await redis.smembersAsync('temps')
    await Promise.all(temps.map(async temp => {
        await fs.unlinkAsync(temp).catch(err =>
            winston.warn('temp: ' + err))
        winston.info('del temp ' + temp)
    }))
    await redis.delAsync('temps')
}
