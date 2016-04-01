/*
Builds client JS and CSS
 */
'use strict';

const babelify = require('babelify'),
	browserify = require('browserify'),
	buffer = require('vinyl-buffer'),
	config = require('./config'),
	gulp = require('gulp'),
	gulpif = require('gulp-if'),
	gutil = require('gulp-util'),
	less = require('gulp-less'),
	nano = require('gulp-cssnano'),
	source = require('vinyl-source-stream'),
	sourcemaps = require('gulp-sourcemaps'),
	uglify = require('gulp-uglify');

const debug = config.DEBUG;

function build(name, b, dest) {
	gulp.task(name, function() {
		return bundler(name, b, dest);
	});
}

function bundler(name, b, dest) {
	// TEMP: Don't minify the client, until we get minification support for ES6
	const canMinify = !debug && name !== 'client';
	return b.bundle()
		// Transform into vinyl stream
		.pipe(source(name + '.js'))
		.pipe(buffer())
		.pipe(sourcemaps.init({loadMaps: true}))
		.pipe(gulpif(canMinify, uglify()))
		.on('error', gutil.log)
		.pipe(sourcemaps.write('./'))
		.pipe(gulp.dest(dest));
}

function buildClient() {
	return browserify({
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
		.require('./client/main', {expose: 'main'});
}

// Main client bundler
{
	const b = buildClient()
		// Transpile ES6 functionality that is not yet supported by the latest
		// stable Chrome and FF to ES5
		.transform(babelify, {
			plugins: [
				'babel-plugin-transform-es2015-classes',
				'transform-es2015-block-scoping',
				'transform-es2015-classes',
				'transform-es2015-destructuring',
				'transform-es2015-object-super',
				'transform-es2015-parameters',
				'transform-es2015-sticky-regex',
				'transform-es2015-unicode-regex',
				'transform-strict-mode'
			]
		})
	build('client', b, './www/js');
}

// Less performant client for older browser compatibility
{
	const b = buildClient().transform(babelify, {
		presets: ['es2015'],
		plugins: ['transform-strict-mode']
	})
	build('legacy', b, './www/js')
}

// Libraries
{
	const b = browserify({
		require: [
			'jquery', 'js-cookie', 'underscore', 'backbone', 'backbone.radio',
			'scriptjs', 'dom4'
		],
		debug: true
	})
		.require('./lib/stack-blur', {expose: 'stack-blur'})
		.require('core-js/es6', {expose: 'core-js'});

	build('vendor', b, './www/js');
}

// Language bundles
gulp.task('lang', function() {
	for (let lang of config.LANGS) {
		const b = browserify({debug: true})
			.require(`./lang/${lang}/common`, {expose: 'lang'});
		bundler(lang, b, './www/js/lang');
	}
});

// Moderation
{
	const b = browserify({
		debug: true,
		bundleExternal: false,
		external: ['main']
	})
		.require('./client/mod', {expose: 'mod'})
		.transform(babelify, {
			presets: ['es2015'],
			plugins: ['transform-strict-mode']
		})

	build('mod', b, './state/');
}

gulp.task('css', function() {
	return gulp.src('./less/*.less')
		.pipe(sourcemaps.init())
		.pipe(less())
		.pipe(nano())
		.pipe(sourcemaps.write('./maps/'))
		.pipe(gulp.dest('./www/css'));
});
