/*
Builds client JS and CSS
 */
'use strict'

const _ = require('underscore'),
	babel = require('gulp-babel'),
	cache = require('gulp-cached'),
	chalk = require('chalk'),
	fs = require('fs-extra'),
	gulp = require('gulp'),
	gulpif = require('gulp-if'),
	gutil = require('gulp-util'),
	less = require('gulp-less'),
	nano = require('gulp-cssnano'),
	rename = require('gulp-rename'),
	sourcemaps = require('gulp-sourcemaps'),
	uglify = require('gulp-uglify')

const langs = fs.readdirSync('./lang')
fs.mkdirsSync('./www/js/vendor')

// Keep script alive and rebuild on file changes
// Triggered with the `-w` flag
const watch = gutil.env.w

// Dependancy tasks for the default tasks
const tasks = langs.slice()

// Client JS files
buildClient('es5')
buildClient('es6')

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
langs.forEach(lang =>
	createTask(lang, `./lang/${lang}/client.js`, src =>
		src
			.pipe(rename({basename: lang}))
			.pipe(sourcemaps.init())
			.pipe(babel({plugins: ['transform-es2015-modules-systemjs']}))
			.pipe(uglify())
			.pipe(sourcemaps.write('./maps'))
			.pipe(gulp.dest('./www/js/lang'))))

// Dependancy libraries
copyVendor([
	'./node_modules/systemjs/dist/system.js',
	'./node_modules/systemjs/dist/system.js.map',
	'./node_modules/dom4/build/dom4.js',
	'./lib/sockjs.js'
])
compileVendor('corejs', 'node_modules/core-js/client/core.js')
compileVendor('js-cookie', 'node_modules/js-cookie/src/js.cookie.js')
compileVendor('underscore', 'node_modules/underscore/underscore.js')
compileVendor('stack-blur', './lib/stack-blur.js')

gulp.task('default', tasks)

/**
 * Builds the client files of the apropriate ECMAScript version
 * @param {string} version
 */
function buildClient(version) {
	createTask(version, './client/**/*.js', src =>
		src
			.pipe(sourcemaps.init())
			.pipe(babel(babelConfig(version)))

			// UglifyJS does not yet fully support ES6, so best not minify
			// to be on the safe side
			.pipe(gulpif(version === 'es5', uglify()))
			.pipe(sourcemaps.write('./maps'))
			.pipe(gulp.dest('./www/js/' + version)))
}

/**
 * Create a new gulp taks and set it to execute on default and incrementally
 * rebuild in watch mode.
 * @param {string} name
 * @param {string} path
 * @param {function} task
 */
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

/**
 * Return a babel configuration object, depending on target ES version
 * @param {string} version
 * @returns {Object}
 */
function babelConfig(version) {
	const base = {
		compact: true,
		comments: false
	}
	if (version === 'es5') {
		return _.extend(base, {
			presets: ['es2015'],
			plugins: [
				'transform-es2015-modules-systemjs'
			]
		})
	}
	return _.extend(base, {
		plugins: [
			'transform-es2015-destructuring',
			'transform-es2015-parameters',
			'transform-es2015-modules-systemjs'
		]
	})
}

/**
 * Copy a dependancy library, minify and generate sourcemaps
 * @param {string} name - Task and output file name
 * @param {string} path - path to file
 */
function compileVendor(name, path) {
	createTask(name, path, src =>
		src
			.pipe(rename({basename: name}))
			.pipe(sourcemaps.init())
			.pipe(uglify())
			.pipe(sourcemaps.write('./maps'))
			.pipe(gulp.dest('./www/js/vendor')))
}

/**
 * Copies a dependancy library from node_modules to the vendor directory
 * @param {string[]} paths - File paths
 */
function copyVendor(paths) {
	for (let path of paths) {
		fs.copySync(
			path,
			'./www/js/vendor/' + _.last(path.split('/')),
			{clobber: true}
		)
	}
}
