var browserify = require('browserify'),
	buffer = require('vinyl-buffer'),
	concat = require('gulp-concat'),
	debug = require('./config').DEBUG,
	deps = require('./deps'),
	gulp = require('gulp'),
	gulpif = require('gulp-if'),
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
		.pipe(less({paths: ['./less/mixins']}))
		.pipe(minifyCSS({rebase: false}))
		.pipe(rev())
		.pipe(gulp.dest('./www/css'))
		.pipe(rev.manifest('css.json'))
		.pipe(gulp.dest('./state'));
});

gulp.task('alpha', function() {
	return browserify(require.resolve('./alpha/main.js'), {
		entry: true,
		// Needed for sourcemaps
		debug: true,
		// Make available outside the bundle with require(). Needed for mod.js.
		require: [
			'jquery',
			'underscore',
			'backbone'
		],
	})
		// Exclude these requires on the client
		.exclude('./config')
		.exclude('../../config')
		.exclude('../../hot')
		.exclude('./lang/')
		.exclude('./server/state')
		.exclude('./imager/config')
		.exclude('./hot')
		.bundle()
		// Transform into vinyl stream
		.pipe(source('alpha.js'))
		.pipe(buffer())
		.pipe(sourcemaps.init({loadMaps: true}))
		.pipe(gulpif(!debug, uglify()))
		.pipe(sourcemaps.write('./'))
		.pipe(gulp.dest('./www/js'));
});

(function() {
	gulper('client', deps.CLIENT_DEPS, './www/js');
	gulper('vendor', deps.VENDOR_DEPS, './www/js');
	gulper('mod', deps.MOD_CLIENT_DEPS, './state');
})();
