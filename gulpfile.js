// Builds client JS, CSS and JSON

'use strict'

const babel = require("gulp-babel"),
	gulp = require('gulp'),
	gutil = require('gulp-util'),
	jsonminify = require('gulp-jsonminify'),
	less = require('gulp-less'),
	minifier = require('gulp-uglify/minifier'),
	nano = require('gulp-cssnano'),
	rename = require('gulp-rename'),
	sourcemaps = require('gulp-sourcemaps'),
	ts = require('gulp-typescript'),
	uglifyjs = require('uglify-js')

// Keep script alive and rebuild on file changes
// Triggered with the `-w` flag
const watch = gutil.env.w

// Dependency tasks for the default tasks
const tasks = []

// Client JS files
buildES6()
buildES5()

// Various little scripts
createTask('scripts', 'clientScripts/*.js', src =>
	src
		.pipe(sourcemaps.init())
		.pipe(minifier({}, uglifyjs))
		.on('error', handleError)
		.pipe(sourcemaps.write('maps'))
		.pipe(gulp.dest('www/js/scripts'))
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
			.pipe(nano())
			.pipe(sourcemaps.write('maps'))
			.pipe(gulp.dest('www/css'))
	)

	// Recompile on source update, if running with the `-w` flag
	if (watch) {
		gulp.watch('less/*.less', () =>
			gulp.start('css'))
	}
}

// Language packs
createTask('lang', 'lang/**/*.json', src =>
	src
		.pipe(jsonminify())
		.on('error', handleError)
		.pipe(gulp.dest('www/lang'))
)

gulp.task('default', tasks)

// Builds the client files of the appropriate ECMAScript version
function buildES6() {
	const name = 'es6'
	tasks.push(name)
	gulp.task(name, () =>
		buildClient()
			.pipe(sourcemaps.write('maps'))
			.pipe(gulp.dest('www/js/es6')))

	// Recompile on source update, if running with the `-w` flag
	if (watch) {
		gulp.watch('client/**/*.ts', [name])
	}
}

// Build legacy ES5 client for old browsers
function buildES5() {
	const name = "es5"
	tasks.push(name)
	gulp.task(name, () =>
		buildClient()
			.pipe(babel({
				presets: ['latest'],
			}))
			.pipe(minifier({}, uglifyjs))
			.on('error', handleError)
			.pipe(sourcemaps.write('maps'))
			.pipe(gulp.dest('www/js/es5'))
	)
}

function buildClient() {
	return gulp.src('client/**/*.ts')
		.pipe(sourcemaps.init())
		.pipe(ts.createProject('client/tsconfig.json', {
			typescript: require("typescript"),
		})())
		.on('error', handleError)
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
