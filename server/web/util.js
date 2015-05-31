/*
 Various utility functions
 */

'use strict';

function parse_forwarded_for(ff) {
	if (!ff)
		return null;
	const ips = ff.split(',');
	if (!ips.length)
		return null;
	const last = ips[ips.length - 1].trim();
	// check that it looks like some kind of IPv4/v6 address
	if (!/^[\da-fA-F.:]{3,45}$/.test(last))
		return null;
	return last;
}
exports.parse_forwarded_for = parse_forwarded_for;
