// Builds client JS, CSS and JSON

'use strict'

const gulp = require('gulp'),
	gutil = require('gulp-util'),
	less = require('gulp-less'),
	cssmin = require('gulp-clean-css'),
	sourcemaps = require('gulp-sourcemaps'),
	ts = require('gulp-typescript'),
	uglify = require('gulp-uglify')

// Keep script alive and rebuild on file changes
// Triggered with the `-w` flag
const watch = gutil.env.w

// Dependency tasks for the default tasks
const tasks = []

// Client JS files
buildClient("frontend")

// Various little scripts
createTask('scripts', 'client/*.js', src =>
	src.pipe(sourcemaps.init())
		.pipe(uglify())
		.on('error', handleError)
		.pipe(sourcemaps.write('maps'))
		.pipe(gulp.dest('www/js'))
)

// Compile Less to CSS
{
	const name = "css"
	tasks.push(name)
	gulp.task(name, () =>
		gulp.src(['less/*.less', '!less/*.mix.less'])
			.pipe(sourcemaps.init())
			.pipe(less())
			.on('error', handleError)
			.pipe(cssmin())
			.pipe(sourcemaps.write('maps'))
			.pipe(gulp.dest('www/css'))
	)

	// Recompile on source update, if running with the `-w` flag
	if (watch) {
		gulp.watch('less/*.less', () =>
			gulp.start('css'))
	}
}

gulp.task('default', tasks)

function buildClient(name) {
	const out = name === "frontend" ? "main" : "worker";
	createTask(name, `client/${name}/**/*.ts`, src =>
		src.pipe(sourcemaps.init())
			.pipe(ts.createProject(`client/${name}/tsconfig.json`, {
				typescript: require("typescript"),
			})())
			.on('error', handleError)
			// .pipe(compiler({
			// 	// Higher levels don't seem to produce valid code
			// 	compilationLevel: 'WHITESPACE_ONLY',
			// 	warningLevel: "QUIET",
			// 	languageIn: "ES6",
			// 	languageOut: "ES6",
			// 	jsOutputFile: `${out}.js`,  // outputs single file
			// 	createSourceMap: true,
			// }))
			.pipe(sourcemaps.write('maps'))
			.pipe(gulp.dest('www/js'))
	)

	// Also watch common client files
	if (watch) {
		gulp.watch("client/common/**/*.ts", [name])
	}
}

// Simply log the error on continuos builds, but fail the build and exit with
// an error status, if failing a one-time build. This way we can use failure to
// build the client to not pass Travis CL tests.
function handleError(err) {
	if (!watch) {
		throw err
	} else {
		console.error(err.message)
	}
}

// Create a new gulp task and set it to execute on default and incrementally
function createTask(name, path, task) {
	tasks.push(name)
	gulp.task(name, () =>
		task(gulp.src(path))
	)

	// Recompile on source update, if running with the `-w` flag
	if (watch) {
		gulp.watch(path, [name])
	}
}
