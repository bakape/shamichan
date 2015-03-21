var browserify = require('browserify'),
	concat = require('gulp-concat'),
	d = require('./config').DEBUG,
	deps = require('./deps'),
	gulp = require('gulp'),
	gulpif = require('gulp-if'),
	less = require('gulp-less'),
	minifyCSS = require('gulp-minify-css'),
	rename = require('gulp-rename'),
	rev = require('gulp-rev'),
	sourcemaps = require('gulp-sourcemaps'),
	transform = require('vinyl-transform'),
	uglify = require('gulp-uglify');

function gulper(name, files, dest) {
	gulp.task(name, function() {
		return gulp.src(files)
			.pipe(concat(name))
			.pipe(gulpif(!d, uglify()))
			.pipe(rev())
			.pipe(rename({suffix: '.' + (d ? 'debug' : 'min') + '.js'}))
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
	// transform regular node stream to gulp (buffered vinyl) stream
	var browserified = transform(function(filename) {
		var b = browserify({entries: filename, debug: true})
			.exclude('./config')
			.exclude('./server/state')
			.exclude('./imager/config');
		return b.bundle();
	});
	return gulp.src('./alpha/main.js')
		.pipe(browserified)
		.pipe(sourcemaps.init({loadMaps: true}))
		.pipe(gulpif(!d, uglify()))
		.pipe(rename({basename: 'alpha'}))
		.pipe(rev())
		.pipe(rename({extname: '.' + (d ? 'debug' : 'min') + '.js'}))
		.pipe(sourcemaps.write('./'))
		.pipe(gulp.dest('./www/js'))
		.pipe(rev.manifest('alpha.json'))
		.pipe(gulp.dest('./state'));
});

(function() {
	gulper('client', deps.CLIENT_DEPS, './www/js');
	gulper('vendor', deps.VENDOR_DEPS, './www/js');
	gulper('mod', deps.MOD_CLIENT_DEPS, './state');
})();
