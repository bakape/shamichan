var concat = require('gulp-concat'),
	d = require('./config').DEBUG,
	deps = require('./deps'),
	gulp = require('gulp'),
	gulpif = require('gulp-if'),
	less = require('gulp-less'),
	minifyCSS = require('gulp-minify-css'),
	rename = require('gulp-rename'),
	rev = require('gulp-rev'),
	uglify = require('gulp-uglify');

function gulper(name, files, dest){
	gulp.task(name, function(){
		gulp.src(files)
			.pipe(concat(name))
			.pipe(gulpif(!d, uglify()))
			.pipe(rev())
			.pipe(rename({ suffix: '.'+(d?'debug':'min')+'.js'}))
			.pipe(gulp.dest(dest))
			.pipe(rev.manifest(name+'.json'))
			.pipe(gulp.dest('./state'));
	});
}

gulp.task('css', function() {
	gulp.src('./less/*.less')
		.pipe(less({paths: ['./less/mixins']}))
		.pipe(minifyCSS({rebase: false}))
		.pipe(rev())
		.pipe(gulp.dest('./www/css'))
		.pipe(rev.manifest('css.json'))
		.pipe(gulp.dest('./state'));
});

(function(){
	gulper('client', deps.CLIENT_DEPS, './www/js');
	gulper('vendor', deps.VENDOR_DEPS, './www/js');
	gulper('mod', deps.MOD_CLIENT_DEPS, './state');
})();
