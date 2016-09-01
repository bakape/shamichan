let cachedOffset;
export function serverTime() :number {
	const d = Date.now();
	if (imports.isNode)
		return d;

	// The offset is intialised as 0, so there is something to return, until
	// we get a propper number from the server.
	if (!cachedOffset)
		cachedOffset = imports.main.request('time:offset');
	return d + cachedOffset;
}

export function parse_name(name) {
	var tripcode = '', secure = '';
	var hash = name.indexOf('#');
	if (hash >= 0) {
		tripcode = name.substr(hash + 1);
		name = name.substr(0, hash);
		hash = tripcode.indexOf('#');
		if (hash >= 0) {
			secure = escape(tripcode.substr(hash + 1));
			tripcode = tripcode.substr(0, hash);
		}
		tripcode = escape(tripcode);
	}
	name = name.trim().replace(imports.hotConfig.EXCLUDE_REGEXP, '');
	return [
		name.substr(0, 100), tripcode.substr(0, 128),
		secure.substr(0, 128)
	];
}
