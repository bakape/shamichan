var config = require('../config');
var fs = require('fs');
var winston = require('winston');

var cssDir = './www/css/';

// Remove previous symlinks
function remove_old(cb){
	fs.readdir(cssDir, function(err, files){
		if (err)
			return winston.error('CSS unlinking error: ' + err);
		files.forEach(function(entry){
			if (/-v\d+.css/.test(entry))
				fs.unlink(cssDir + entry);
		});
		cb();
	});
}

function linker(source, version){
	fs.realpath(cssDir + source + '.css', function(err, path){
		if (err)
			return winston.error('CSS realpath error: ' + err);
		fs.symlinkSync(path, cssDir + source + '-v' + version + '.css');
	});	
}

(function(){
	remove_old(function(){
		// Generate new symlinks
		config.THEMES.forEach(function(entry){
			linker(entry, config.THEME_CSS_VERSION);
		});
		linker('base', config.BASE_CSS_VERSION);
		linker('mod', config.MOD_CSS_VERSION);
		linker('gravitas', config.GRAVITAS_CSS_VERSION);
	});
})();
