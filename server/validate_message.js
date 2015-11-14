/*
 Verify websocket messages confirm to a predefined type schema
 */

/**
 * Validate an object conforms to the provided schema
 * @param {Object} schema
 * @param {*} msg
 * @returns {boolean}
 */
function object(schema, msg) {
	if (typeof msg !== 'object' || msg === null || msg instanceof Array)
		return false
	for (let key in schema) {
		let spec = schema[key]

		// Optional key
		if (spec.startsWith('opt ')) {
			if (!(key in msg))
				continue
			spec = spec.slice(4)
		}
		// Manditory key
		else if (!(key in msg))
			return false

		if (!value(spec, msg[key]))
			return false
	}
	return true
}
exports.object = object

/**
 * Validate a value is of the specified type
 * @param {string} spec
 * @param {*} val
 * @returns {boolean}
 */
function value(spec, val) {
	switch (spec) {
		case 'id':
			return typeof val === 'number' && Number.isInteger(val) && val >= 1
		case 'string':
		case 'boolean':
			return typeof val === spec
	}
}
exports.value = value

/**
 * Validates a fixed length array againsta schema
 * @param {Array} schema
 * @param {*} msg
 * @returns {boolean}
 */
function array(schema, msg) {
	if (!(msg instanceof Array) || msg.length !== schema.length)
		return false
	for (let i = 0; i < schema.length; i++) {
		if (!value(schema[i], msg[i]))
			return false
	}
	return true
}
exports.array = array

/**
 * Validate a dynamic length array only contains members of specific type
 * @param {string} spec
 * @param {*} msg
 * @returns {boolean}
 */
function dynamicArray(spec, msg) {
	if (!(msg instanceof Array))
		return false
	for (let item of msg) {
		if (!value(spec, item))
			return false
	}
	return true
}
exports.dynamicArray = dynamicArray
