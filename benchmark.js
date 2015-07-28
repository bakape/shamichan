/*
Basic synchronous and asynchronous benchmarks for individual functional units
 */
'use strict';

// Usage examples
let number = 0;
benchmark('Addition', 1000, function () {
	number++;
});

let redis = require('redis').createClient();
asyncBenchmark('Redis ping', 1000, function (cb) {
	redis.ping(cb);
});

function benchmark(name, iterations, func) {
	console.time(name);
	for (let i = 0; i < iterations; i++) {
		func();
	}
	console.timeEnd(name);
}

function asyncBenchmark(name, iterations, func, next) {
	console.time(name);
	let i = 0;
	loop();

	function loop() {
		// The benchmarked function must take a callback as its only argument
		func(++i < iterations ? loop : finish);
	}

	function finish() {
		console.timeEnd(name);
		// Execute next async benchmark
		if (next)
			next();
		else
			process.exit();
	}
}
