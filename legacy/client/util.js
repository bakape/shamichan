let cachedOffset;
export function serverTime() {
	const d = Date.now();
	if (imports.isNode)
		return d;

	// The offset is intialised as 0, so there is something to return, until
	// we get a propper number from the server.
	if (!cachedOffset)
		cachedOffset = imports.main.request('time:offset');
	return d + cachedOffset;
}
