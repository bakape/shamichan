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
gulp.task('default', tasks)

// Main client bundles
clientBundles('main',
	// Make available outside the bundle with require() under a
	// shorthand name
	['./client/main/main', {expose: 'main'}],
	browserifyOpts({
		entries: './client/main/main',
		bundleExternal: false,
		external: [
			'js-cookie', 'underscore', 'backbone', 'backbone.radio',
			'stack-blur', 'lang', 'core-js', 'scriptjs', 'dom4',
			"backbone.nativeview", "main"
		]
	}))

// Libraries
createTask('vendor', 'www/js/vendor', true, browserifyOpts({
		require: [
			'js-cookie', 'underscore', 'backbone', 'backbone.radio',
			'scriptjs', 'sockjs-client', 'dom4', 'backbone.nativeview'
		]
	})
		.exclude('jquery')
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

// Recompile on source update
if (watch) {
	gulp.watch('./client/scripts/*.js', ['scripts'])
}

// Moderation bundles
clientBundles('mod', ['./client/mod', {expose: 'mod'}], browserifyOpts({
		bundleExternal: false,
		external: ['main']
	}))

// Compile Less to CSS
gulp.task('css', () =>
	gulp.src('./less/*.less')
		.on('error', gutil.log)
		.pipe(sourcemaps.init())
		.pipe(less())
		.pipe(nano())
		.pipe(sourcemaps.write('./maps/'))
		.pipe(gulp.dest('./www/css')))

if (watch) {
	gulp.watch('./less/*.less', ['css'])
}

/**
 * Merge custom browserify options with common ones
 */
function browserifyOpts(opts) {
    const base = {debug: true} // Needed for sourcemaps
	if (watch) {
		_.extend(base, {
			cache: {},
			packageCache: {},
			plugin: [watchify]
		})
	}
	return browserify(_.extend(base, opts))
}

/**
 * Build a client JS bundle
 */
function clientBundles(name, requires, b) {
	const versions = {
		es5: compileES5(b),
		es6: compileES6(b)
	}
	for (let key in versions) {
	    versions[key] = versions[key].require(...requires)
	}
	createTask(name + '.es5', 'www/js/es5', true, versions.es5)
	createTask(name + '.es6', 'www/js/es6', false, versions.es6)
}

/**
 * Create a gulp task for compiling JS
 */
function createTask(name, dest, es5, b) {
    gulp.task(name, () => {
		if (watch)
			b.on("update", run.bind(null, true))
		return run()

		function run(rebuild) {
		    recompileLog(rebuild, true, name)
		    return bundle(name, dest, es5, rebuild, b)
		}
	})
}

/**
 * Create a single bundle, process and write it to disk
 */
function bundle(name, dest, es5, rebuild, b) {
	return b.bundle()
		// Browserify error logging
		.on('error', function (err) {
			console.error(err.stack)
			this.emit('end')
	    })
		// Transform into vinyl stream for Browserify compatibility with gulp
		.pipe(source(name.replace(/\.es\d/, '') + '.js'))
		.pipe(buffer())
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
		plugins: ['transform-strict-mode'],
		compact: true,
		comments: false
	})
}

/**
 * Compile ES6 functionality that is not yet supported by the latest stable
 * Chrome and Firefox to ES5
 */
function compileES6(b) {
    return b.transform(babelify, {
		plugins: [
			'transform-es2015-destructuring',
			'transform-strict-mode',
			'transform-es2015-modules-commonjs'
		],
		compact: true,
		comments: false
	})
}
