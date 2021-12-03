// Builds client JS, CSS and JSON

'use strict'

const gulp = require('gulp'),
	less = require('gulp-less'),
	cssmin = require('gulp-clean-css'),
	ts = require('gulp-typescript'),
	uglify = require('gulp-uglify')

// Dependency tasks for the default tasks
const tasks = []

// Client JS files
createTask("client", `client/**/*.ts`, src =>
	src
		.pipe(ts.createProject("client/tsconfig.json", {
			typescript: require("typescript"),
		})())
		.on('error', handleError)
		.pipe(gulp.dest('www/js'))
)

createTask("static", "clientStatic/*.ts", src =>
	src
		.pipe(ts.createProject("client/tsconfig.json", {
			typescript: require("typescript"),
		})())
		.on("error", handleError)
		.pipe(gulp.dest("www/js/static"))
)

// Various little scripts
createTask('scripts', 'clientScripts/*.js', src =>
	src
		.pipe(uglify())
		.on('error', handleError)
		.pipe(gulp.dest('www/js/scripts'))
)

// Compile Less to CSS
{
	const name = "css"
	tasks.push(name)
	gulp.task(name, () =>
		gulp.src(['less/*.less', '!less/*.mix.less'])
			.pipe(less())
			.on('error', handleError)
			.pipe(cssmin())
			.pipe(gulp.dest('www/css'))
	)
}

exports.default = gulp.parallel(...tasks);

// Simply log the error on continuos builds, but fail the build and exit with
// an error status, if failing a one-time build. This way we can use failure to
// build the client to not pass Travis CL tests.
function handleError(err) {
	console.error(err.message)
}

// Create a new gulp task and set it to execute on default and incrementally
function createTask(name, path, task) {
	tasks.push(name)
	gulp.task(name, () =>
		task(gulp.src(path))
	)
}
