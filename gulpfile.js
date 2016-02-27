/*
Builds client JS and CSS
 */
'use strict'

const _ = require('underscore'),
	babel = require('gulp-babel'),
	cache = require('gulp-cached'),
	fs = require('fs-extra'),
	gulp = require('gulp'),
	gutil = require('gulp-util'),
	jsonminify = require('gulp-jsonminify'),
	less = require('gulp-less'),
	nano = require('gulp-cssnano'),
	rename = require('gulp-rename'),
	sourcemaps = require('gulp-sourcemaps'),
	ts = require('gulp-typescript'),
	uglify = require('gulp-uglify')

fs.mkdirsSync('./www/js/vendor')

// Keep script alive and rebuild on file changes
// Triggered with the `-w` flag
const watch = gutil.env.w

// Dependancy tasks for the default tasks
const tasks = []

// Client JS files
buildClient()

// Various little scripts
createTask('scripts', './clientScripts/*.js', src =>
	src
		.pipe(sourcemaps.init())
		.pipe(uglify())
		.pipe(sourcemaps.write('./maps'))
		.pipe(gulp.dest('./www/js/scripts')))

// Compile Less to CSS
createTask('css', './less/*.less', src =>
	src
		.pipe(sourcemaps.init())
		.pipe(less())
		.pipe(nano())
		.pipe(sourcemaps.write('./maps'))
		.pipe(gulp.dest('./www/css')))

// Language packs
createTask('lang', './lang/*.json', src =>
	src
		.pipe(jsonminify())
		.pipe(gulp.dest('./www/lang')))

// Dependancy libraries
copyVendor([
	'./node_modules/systemjs/dist/system.js',
	'./node_modules/systemjs/dist/system.js.map',
	'./node_modules/dom4/build/dom4.js'
])
compileVendor('underscore', 'node_modules/underscore/underscore.js')
compileVendor('stack-blur', './lib/stack-blur.js')

gulp.task('default', tasks)

const tsProject = ts.createProject('./client/tsconfig.json')

// Builds the client files of the apropriate ECMAScript version
function buildClient() {
	createTask("client", './client/**/*.ts', src =>
		src
			.pipe(sourcemaps.init())
			.pipe(ts(tsProject))
			.pipe(babel({
				compact: true,
				comments: false,
				plugins: [
					'transform-es2015-destructuring',
					'transform-es2015-parameters'
				]
			}))
			.pipe(sourcemaps.write('./maps'))
			.pipe(gulp.dest('./www/js/')))
}

// Create a new gulp taks and set it to execute on default and incrementally
function createTask(name, path, task) {
	tasks.push(name)
	gulp.task(name, () =>
		task(gulp.src(path)
			.on('error', gutil.log)
			.pipe(cache(name))))

	// Recompile on source update, if running with the `-w` flag
	if (watch) {
		gulp.watch(path, [name])
	}
}

// Copy a dependancy library, minify and generate sourcemaps
function compileVendor(name, path) {
	createTask(name, path, src =>
		src
			.pipe(rename({basename: name}))
			.pipe(sourcemaps.init())
			.pipe(uglify())
			.pipe(sourcemaps.write('./maps'))
			.pipe(gulp.dest('./www/js/vendor')))
}

// Copies a dependancy library from node_modules to the vendor directory
function copyVendor(paths) {
	for (let path of paths) {
		fs.copySync(
			path,
			'./www/js/vendor/' + _.last(path.split('/')),
			{clobber: true}
		)
	}
}
