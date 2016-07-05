/*
Core server module and application entry point
 */

const dispatcher = websockets.dispatcher;

/**
 * Validate post has the proper fields and client has posting rights
 * @param {Object} msg
 * @param {Object} spec
 * @param {Client} client
 * @returns {boolean}
 */
function canInsertPost(msg, spec, client) {
    const {frag, image} = msg
    return !config.READ_ONLY
        && caps.can_access_board(client.ident, client.board)
        && validate.object(spec, msg)
        && (frag || image)
        && !(frag && /^\s*$/g.test(frag))
}

dispatcher[common.INSERT_POST] = ([msg], client) => {
	const spec = {
		frag: 'opt string',
		image: 'opt string',
		nonce: 'string',
		name: 'opt string',
		email: 'opt string',
		auth: 'opt string'
	}
	if (!canInsertPost(msg, spec, client))
		return false
	client.db.insertPost(msg).catch(err =>
		client.disconnect(Muggle('Allocation failure', err)))
	return true
}

dispatcher[common.UPDATE_POST] = (frag, client) => {
	if (typeof frag !== 'string')
		return false
	frag = amusement.hot_filter(frag.replace(STATE.hot.EXCLUDE_REGEXP, ''))
	const {post} = client
	if (!post)
		return false
	const limit = common.MAX_POST_CHARS
	if (frag.length > limit || client.postLength  >= limit)
		return false
	const combined = client.postLength + frag.length
	if (combined > limit)
		frag = frag.substr(0, combined - limit)
	client.db.appendPost(frag).catch(err =>
		client.disconnect(Muggle("Couldn't add text.", err)))
	return true
}

dispatcher[common.FINISH_POST] = ([msg], client) => {
    if (typeof msg !== 'string')
        return false
    client.db.finishPost().catch(err =>
        client.disconnect(Muggle("Couldn't finish post", err)))
	return true
}

dispatcher[common.INSERT_IMAGE] = function ([msg], client) {
	if (typeof msg !== 'string' || !client.post || client.post.image)
		return false
	client.db.insertImage(msg).catch(err =>
		client.disconnect(Muggle('Image insertion error:', err)))
	return true
}
