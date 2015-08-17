'use strict';

let babelify = require('babelify'),
	browserify = require('browserify'),
	buffer = require('vinyl-buffer'),
	config = require('./config'),
	gulp = require('gulp'),
	gulpif = require('gulp-if'),
	gutil = require('gulp-util'),
	less = require('gulp-less'),
	minifyCSS = require('gulp-minify-css'),
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
			'jquery',
			'js-cookie',
			'underscore',
			'backbone',
			'backbone.radio',
			'stack-blur',
			'lang',
			'core-js',
			'scriptjs'
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
	let b = buildClient()
		// Transpile ES6 functionality that is not yet supported by the latest
		// stable Chrome and FF to ES5. Ancient and hipster browsers can
		// suck my dick.
		.transform(babelify.configure({
			// MUH PERFORMINCE
			blacklist: [
				'es3.memberExpressionLiterals',
				'es3.propertyLiterals',
				'es5.properties.mutators',
				'es6.constants',
				'es6.forOf',
				'es6.properties.computed',
				'es6.properties.shorthand',
				'es6.spec.templateLiterals',
				'es6.templateLiterals',
				'flow',
				'react',
				'jscript',
				'react',
				'reactCompat',
				'regenerator',
				'runtime'
			]
		}));

	build('client', b, './www/js');
}

// Less performant client for older browser compatibility
{
	let b = buildClient().transform(babelify.configure({
		optional: ['es6.spec.blockScoping']
	}));

	build('legacy', b, './www/js');
}

// Libraries
{
	let b = browserify({
		require: [
			'jquery',
			'js-cookie',
			'underscore',
			'backbone',
			'backbone.radio',
			'scriptjs',
			'sockjs-client'
		],
		debug: true
	})
		.require('./lib/stack-blur', {expose: 'stack-blur'})
		.require('core-js/es6', {expose: 'core-js'});

	build('vendor', b, './www/js');
}

// Language bundles
gulp.task('lang', function() {
	const langs = config.LANGS;
	for (let i = 0, l = langs.length; i < l; i++) {
		const lang = langs[i];
		let b = browserify({debug: true})
			.require(`./lang/${lang}/common`, {expose: 'lang'});
		bundler(lang, b, './www/js/lang');
	}
});

// Moderation
{
	let b = browserify({
		debug: true,
		bundleExternal: false,
		external: ['main']
	})
		.require('./admin/client', {expose: 'mod'})
		.transform(babelify.configure({
			optional: ['es6.spec.blockScoping']
		}));

	build('mod', b, './state/');
}

gulp.task('css', function() {
	return gulp.src('./less/*.less')
		.pipe(sourcemaps.init())
		.pipe(less())
		.pipe(minifyCSS({rebase: false}))
		.pipe(sourcemaps.write('./maps/'))
		.pipe(gulp.dest('./www/css'));
});
