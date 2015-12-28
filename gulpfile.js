/*
Builds client JS and CSS
 */
'use strict'

const _ = require('underscore'),
	babelify = require('babelify'),
	browserify = require('browserify'),
	buffer = require('vinyl-buffer'),
	chalk = require('chalk'),
	fs = require('fs'),
	gulp = require('gulp'),
	gulpif = require('gulp-if'),
	gutil = require('gulp-util'),
	less = require('gulp-less'),
	nano = require('gulp-cssnano'),
	source = require('vinyl-source-stream'),
	sourcemaps = require('gulp-sourcemaps'),
	watchify = require('watchify'),
	uglify = require('gulp-uglify')

const langs = fs.readdirSync('./lang'),
	// Keep script alive and rebuild on file changes
	// Triggered with the --watch flag
	watch = gutil.env.watch

// Shorthand for compiling everything with no task arguments
const tasks = ['vendor', 'css', 'scripts'].concat(langs)
; ['main', 'mod'].forEach(name =>
	tasks.push(name + '.es5', name + '.es6'))
gulp.task('default', tasks, () =>
	!watch && process.exit(0))

// Main client bundles
clientBundles('main',
	browserifyOpts({
		entries: './client/main',
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
createTask('vendor', 'www/js/vendor', true, browserifyOpts({
		require: [
			'jquery', 'js-cookie', 'underscore', 'backbone', 'backbone.radio',
			'scriptjs', 'sockjs-client', 'dom4'
		]
	})
		.require('./lib/stack-blur', {expose: 'stack-blur'})
		.require('core-js/es6', {expose: 'core-js'}))

// Language bundles
langs.forEach(lang =>
	createTask(lang, 'www/js/lang', true, browserifyOpts({})
		.require(`./lang/${lang}/client`, {expose: 'lang'})))

// Various little scripts
gulp.task('scripts', () =>
	gulp.src('./client/scripts/*.js')
		.on('error', gutil.log)
		.pipe(sourcemaps.init())
		.pipe(uglify())
		.pipe(sourcemaps.write('./'))
		.pipe(gulp.dest('./www/js')))

// Moderation bundles
clientBundles('mod', browserifyOpts({
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
 * Merge custom browserify options with common ones
 */
function browserifyOpts(opts) {
    const base = {
		debug: true, // Needed for sourcemaps
		cache: {},
	    packageCache: {},
	    plugin: [watchify]
	}
	return browserify(_.extend(base, opts))
}

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
    gulp.task(name, () => {
		if (watch)
			b.on("update", run.bind(null, true))
		run()

		function run(rebuild) {
		    recompileLog(rebuild, true, name)
		    bundle(name, dest, es5, rebuild, b)
		}
	})
}

/**
 * Create a single bundle, process and write it to disk
 */
function bundle(name, dest, es5, rebuild, b) {
	return b.bundle()
		// Transform into vinyl stream for Browserify compatibility with gulp
		.pipe(source(name.replace(/\.es\d/, '') + '.js'))
		.pipe(buffer())
		.on('error', gutil.log)
		.on('end', () =>
			recompileLog(rebuild, false, name))
		.pipe(sourcemaps.init({loadMaps: true}))

		// UglifyJS does not yest fully support ES6, so best not minify to be
		// on the safe side
		.pipe(gulpif(es5, uglify()))
		.pipe(sourcemaps.write('./'))
		.pipe(gulp.dest(dest))
}

/**
 * Prints a message on recompilation from watched file updates
 */
function recompileLog(print, starting, name) {
	if (print)
	    gutil.log((starting ? "Recompiling " : "Finished "), chalk.cyan(name))
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
