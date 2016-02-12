/*
Basic synchronous and asynchronous benchmarks for individual functional units
 */
'use strict';

function sync(name, iterations, func) {
	console.time(name);
	for (let i = 0; i < iterations; i++) {
		func();
	}
	console.timeEnd(name);
}
exports.sync = sync;

function async(name, iterations, func, next) {
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
exports.async = async;
