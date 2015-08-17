/*
 It is not very efficient to spam liteners to the options object. This
 module loops through the post models and calls the appropriate methods in
 batch.
 */

const main = require('./main'),
	{etc, options} = main,
	{posts} = main.state;

options.on({
	'change:thumbs': reRenderImages,
	'change:spoilers': toggleSpoilers,
	'change:autogif': toggleAutoGIF,
	'change:anonymise': toggleAnonymisation
});

function reRenderImages() {
	posts.each(function(model) {
		const image = model.get('image');
		if (image)
			model.dispatch('renderImage', image)
	});
}
main.reply('loop:images', reRenderImages);

function toggleSpoilers() {
	posts.each(function(model) {
		const image = model.get('image');
		if (image && image.spoiler)
			model.dispatch('renderImage', image);
	});
}

// Toggle animated GIF thumbnails
function toggleAutoGIF() {
	posts.each(function(model) {
		const image = model.get('image');
		if (image && image.ext === '.gif')
			model.dispatch('renderImage', image);
	});
}

function toggleAnonymisation(source, toggle) {
	const command = toggle ? 'anonymise' : 'renderName';
	posts.each(function(model) {
		const {name, trip} = model.attributes;
		if (name || trip)
			model.dispatch(command);
	});
}
main.reply('loop:anonymise', function() {
	if (options.get('anonymise'))
		toggleAnonymisation(null, true);
});
