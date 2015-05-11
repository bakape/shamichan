var babelify = require('babelify'),
	browserify = require('browserify'),
	buffer = require('vinyl-buffer'),
	concat = require('gulp-concat'),
	debug = require('./config').DEBUG,
	deps = require('./deps'),
	gulp = require('gulp'),
	gulpif = require('gulp-if'),
	gutil = require('gulp-util'),
	less = require('gulp-less'),
	minifyCSS = require('gulp-minify-css'),
	rename = require('gulp-rename'),
	rev = require('gulp-rev'),
	source = require('vinyl-source-stream'),
	sourcemaps = require('gulp-sourcemaps'),
	uglify = require('gulp-uglify');

function gulper(name, files, dest) {
	gulp.task(name, function() {
		return gulp.src(files)
			.pipe(concat(name))
			.pipe(gulpif(!debug, uglify()))
			.pipe(rev())
			.pipe(rename({suffix: '.' + (debug ? 'debug' : 'min') + '.js'}))
			.pipe(gulp.dest(dest))
			.pipe(rev.manifest(name + '.json'))
			.pipe(gulp.dest('./state'));
	});
}

gulp.task('css', function() {
	return gulp.src('./less/*.less')
		.pipe(sourcemaps.init())
		.pipe(less())
		.pipe(minifyCSS({rebase: false}))
		.pipe(rev())
		.pipe(sourcemaps.write('./maps/'))
		.pipe(gulp.dest('./www/css'))
		.pipe(rev.manifest('css.json'))
		.pipe(gulp.dest('./state'));
});

function build(name, b) {
	gulp.task(name, function() {
		return b.bundle()
			// Transform into vinyl stream
			.pipe(source(name + '.js'))
			.pipe(buffer())
			.pipe(sourcemaps.init({loadMaps: true}))
			.pipe(gulpif(!debug, uglify()))
			.on('error', gutil.log)
			.pipe(sourcemaps.write('./'))
			.pipe(gulp.dest('./www/js'));
	});
}

build('client', browserify(require.resolve('./client/main.js'),
	{
		entry: true,
		// Needed for sourcemaps
		debug: true,
		bundleExternal: false,
		external: [
			'jquery',
			'jquery.cookie',
			'underscore',
			'backbone'
		]
	})
		/*
		 Trasnpile to ES5. We use already implemented native functionality, where
		 possible. These are mainly to prevent the minifier from throwing errors
		 and some syntactic sugar, that is not yet implemented. Nobody cares about
		 your ancient retarded browser.
		 */
		.transform(babelify.configure({
			whitelist: [
				'es6.blockScoping',
				'es6.arrowFunctions',
				'es6.parameters.default',
				'es6.parameters.rest',
				'es6.spread',
				'es6.properties.computed',
				'es6.properties.shorthand',
				'es6.spec.templateLiterals',
				'es6.templateLiterals',
				'strict'
			]
		}))
		// Exclude these requires on the client
		.exclude('../config')
		.exclude('../lang/')
		.exclude('../server/state')
);

build('vendor', browserify({
	// Make available outside the bundle with require()
	require: [
		'jquery',
		'jquery.cookie',
		'underscore',
		'backbone'
	],
	debug: true
}));

(function() {
	gulper('mod', deps.mod, './state');
})();
