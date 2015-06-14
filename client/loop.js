/*
 It is not very efficient to spam liteners to the options object. This
 module loops through the post models and calls the appropriate methods in
 batch.
 */

let main = require('./main'),
	{etc, options, state} = main;

let models = state.posts.models;

options.on({
	'change:thumbs': reRenderImages,
	'change:spoilers': toggleSpoilers,
	'change:autogif': toggleAutoGIF,
	'change:anonymise': toggleAnonymisation
});

function reRenderImages() {
	etc.defferLoop(models, function(model) {
		let image = model.get('image');
		if (image)
			model.dispatch('renderImage', image)
	});
}
main.comply('loop:images', () => reRenderImages());

function toggleSpoilers() {
	etc.defferLoop(models, function(model) {
		let image = model.get('image');
		if (image && image.spoiler)
			model.dispatch('renderImage', image);
	});
}

// Toggle animated GIF thumbnails
function toggleAutoGIF() {
	etc.defferLoop(models, function(model) {
		let image = model.get('image');
		if (image && image.ext === '.gif')
			model.dispatch('renderImage', image);
	});
}

function toggleAnonymisation(source, toggle) {
	const command = toggle ? 'anonymise' : 'renderName';
	etc.defferLoop(models, function(model) {
		const {name, trip} = model.attributes;
		if (name || trip)
			model.dispatch(command);
	});
}
main.comply('loop:anonymise', function() {
	if (options.get('anonymise'))
		toggleAnonymisation(null, true);
});
