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

exports.map = function (array, func, callback) {
	var results = [];
	step(0);
	function step(i) {
		if (i >= array.length)
			return callback(null, results);
		func(array[i], function (err, res) {
			if (err)
				return callback(err);
			results.push(res);
			setTimeout(step.bind(null, i + 1), 0);
		});
	}
};
