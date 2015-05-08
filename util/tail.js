// avoids stack overflow for long lists
// TEMP: Remove when tail call optimisation hits io.js
function forEach (array, func, callback) {
	step(0);
	function step(i) {
		if (i >= array.length)
			return callback(null);
		func(array[i], function (err) {
			if (err)
				return callback(err);
			setImmediate(step, i + 1);
		});
	}
}
exports.forEach = forEach;

function map (array, func, callback) {
	var results = [];
	step(0);
	function step(i) {
		if (i >= array.length)
			return callback(null, results);
		func(array[i], function (err, res) {
			if (err)
				return callback(err);
			results.push(res);
			setImmediate(step, i + 1);
		});
	}
}
exports.map = map;
