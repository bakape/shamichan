// avoids stack overflow for long lists
exports.forEach = function (array, func, callback) {
	step(0);
	function step(i) {
		if (i >= array.length)
			return callback(null);
		func(array[i], function (err) {
			if (err)
				return callback(err);
			setTimeout(step.bind(null, i + 1), 0);
		});
	}
};
