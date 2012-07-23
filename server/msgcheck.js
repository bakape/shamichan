
function check(schema, msg) {
	/* Primitives */
	if (schema == 'id' || schema == 'nat')
		return typeof msg == 'number' && (msg || msg === 0) &&
				msg >= (schema == 'id' ? 1 : 0) &&
				Math.round(msg) === msg;
	else if (schema == 'string')
		return typeof msg == 'string';
	else if (schema == 'boolean')
		return typeof msg == 'boolean';

	/* Arrays */
	if (schema instanceof Array) {
		if (!(msg instanceof Array) || msg.length != schema.length)
			return false;
		for (var i = 0; i < schema.length; i++)
			if (!check(schema[i], msg[i]))
				return false;
		return true;
	}
	else if (schema == 'id...') {
		if (!(msg instanceof Array) || !msg.length)
			return false;
		return msg.every(check.bind(null, 'id'));
	}
	else if (msg instanceof Array)
		return false;

	/* Hashes */
	if (typeof schema == 'object') {
		if (typeof msg != 'object' || msg instanceof Array)
			return false;
		for (var k in schema) {
			var spec = schema[k];
			/* optional key */
			if (typeof spec == 'string' && spec.match(/^opt /)) {
				if (!(k in msg))
					continue;
				spec = spec.slice(4);
			}
			else if (!(k in msg))
				return false; /* otherwise mandatory */

			if (!check(spec, msg[k]))
				return false;
		}
		return true;
	}
	else if (schema == 'id=>nat') {
		if (typeof msg != 'object' || msg instanceof Array)
			return false;
		for (var k in msg) {
			if (!k.match(/^[1-9]\d*$/))
				return false;
			if (!check('nat', msg[k]))
				return false;
		}
		return true;
	}

	throw new Error("Unknown schema: " + schema);
}

exports.check = check;
