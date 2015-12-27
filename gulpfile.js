/*
Builds client JS and CSS
 */
'use strict'

const babelify = require('babelify'),
	browserify = require('browserify'),
	buffer = require('vinyl-buffer'),
	config = require('./config/config.json'),
	gulp = require('gulp'),
	gulpif = require('gulp-if'),
	gutil = require('gulp-util'),
	less = require('gulp-less'),
	nano = require('gulp-cssnano'),
	source = require('vinyl-source-stream'),
	sourcemaps = require('gulp-sourcemaps'),
	uglify = require('gulp-uglify')

const debug = config.hard.debug

// Shorthand for compiling everything with no task arguments
const tasks = ['vendor', 'css', 'scripts'].concat(config.lang.enabled)
; ['main', 'mod'].forEach(name =>
	tasks.push(name + '.es5', name + '.es6'))
gulp.task('default', tasks)

// Main client bundles
clientBundles('main',
	browserify({
		entries: './client/main',
		// Needed for sourcemaps
		debug: true,
		bundleExternal: false,
		external: [
			'jquery', 'js-cookie', 'underscore', 'backbone', 'backbone.radio',
			'stack-blur', 'lang', 'core-js', 'scriptjs', 'dom4'
		]
	})
		// Exclude these requires on the client
		.exclude('../config')
		.exclude('../lang/')
		.exclude('../server/state')
		// Make available outside the bundle with require() under a
		// shorthand name
		.require('./client/main', {expose: 'main'}))

// Libraries
createTask('vendor', 'www/js/vendor', true, browserify({
		require: [
			'jquery', 'js-cookie', 'underscore', 'backbone', 'backbone.radio',
			'scriptjs', 'sockjs-client', 'dom4'
		],
		debug: true
	})
		.require('./lib/stack-blur', {expose: 'stack-blur'})
		.require('core-js/es6', {expose: 'core-js'}))

// Language bundles
config.lang.enabled.forEach(lang =>
	createTask(lang, 'www/js/lang', true, browserify({debug: true})
		.require(`./lang/${lang}/client`, {expose: 'lang'})))

// Various little scripts
gulp.task('scripts', () =>
	gulp.src('./client/scripts/*.js')
		.pipe(sourcemaps.init())
		.pipe(gulpif(!debug, uglify()))
		.on('error', gutil.log)
		.pipe(sourcemaps.write('./'))
		.pipe(gulp.dest('./www/js')))

// Moderation bundles
clientBundles('mod', browserify({
		debug: true,
		bundleExternal: false,
		external: ['main']
	})
		.require('./client/mod', {expose: 'mod'}))

// Compile Less to CSS
gulp.task('css', () =>
	gulp.src('./less/*.less')
		.on('error', gutil.log)
		.pipe(sourcemaps.init())
		.pipe(less())
		.pipe(nano())
		.pipe(sourcemaps.write('./maps/'))
		.pipe(gulp.dest('./www/css')))

/**
 * Build a client JS bundle
 */
function clientBundles(name, b) {
	createTask(name + '.es5', 'www/js/es5', true, compileES5(b))
	createTask(name + '.es6', 'www/js/es6', false, compileES6(b))
}

/**
 * Create a gulp task for compiling JS
 */
function createTask(name, dest, es5, b) {
    gulp.task(name, () =>
		bundle(name, dest, es5, b))
}

/**
 * Create a single bundle, process and write it to disk
 */
function bundle(name, dest, es5, b) {
	return b.bundle()
		// Transform into vinyl stream for Browserify compatibility with gulp
		.pipe(source(name.replace(/\.es\d/, '') + '.js'))
		.pipe(buffer())
		.on('error', gutil.log)
		.pipe(sourcemaps.init({loadMaps: true}))

		// UglifyJS does not yest fully support ES6, so best not minify to be
		// on the safe side
		.pipe(gulpif(es5 && !debug, uglify()))
		.pipe(sourcemaps.write('./'))
		.pipe(gulp.dest(dest))
}

/**
 * Compile to pure ES5 for older browsers
 */
function compileES5(b) {
    return b.transform(babelify, {
		presets: ['es2015'],
		plugins: ['transform-strict-mode']
	})
}

/**
 * Compile ES6 functionality that is not yet supported by the latest stable
 * Chrome and Firefox to ES5
 */
function compileES6(b) {
    return b.transform(babelify, {
		plugins: [
			'babel-plugin-transform-es2015-classes',
			'transform-es2015-block-scoping',
			'transform-es2015-classes',
			'transform-es2015-destructuring',
			'transform-es2015-object-super',
			'transform-strict-mode',
			'transform-es2015-modules-commonjs'
		]
	})
}
