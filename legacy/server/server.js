dispatcher[common.INSERT_IMAGE] = function ([msg], client) {
	if (typeof msg !== 'string' || !client.post || client.post.image)
		return false
	client.db.insertImage(msg).catch(err =>
		client.disconnect(Muggle('Image insertion error:', err)))
	return true
}
