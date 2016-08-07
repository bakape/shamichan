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

dispatcher[common.INSERT_IMAGE] = function ([msg], client) {
	if (typeof msg !== 'string' || !client.post || client.post.image)
		return false
	client.db.insertImage(msg).catch(err =>
		client.disconnect(Muggle('Image insertion error:', err)))
	return true
}
