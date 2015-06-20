/*
 It is not very efficient to spam liteners to the options object. This
 module loops through the post models and calls the appropriate methods in
 batch.
 */

let main = require('./main'),
	{etc, options, state} = main;

options.on({
	'change:thumbs': reRenderImages,
	'change:spoilers': toggleSpoilers,
	'change:autogif': toggleAutoGIF,
	'change:anonymise': toggleAnonymisation
});

function reRenderImages() {
	loop(function(model) {
		let image = model.get('image');
		if (image)
			model.dispatch('renderImage', image)
	});
}
main.comply('loop:images', () => reRenderImages());

function loop(func) {
	// Shallow copy array to remove refference
	etc.deferLoop(state.posts.models.slice(), 1, func);
}

function toggleSpoilers() {
	loop(function(model) {
		let image = model.get('image');
		if (image && image.spoiler)
			model.dispatch('renderImage', image);
	});
}

// Toggle animated GIF thumbnails
function toggleAutoGIF() {
	loop(function(model) {
		let image = model.get('image');
		if (image && image.ext === '.gif')
			model.dispatch('renderImage', image);
	});
}

function toggleAnonymisation(source, toggle) {
	const command = toggle ? 'anonymise' : 'renderName';
	loop(function(model) {
		const {name, trip} = model.attributes;
		if (name || trip)
			model.dispatch(command);
	});
}
main.comply('loop:anonymise', function() {
	if (options.get('anonymise'))
		toggleAnonymisation(null, true);
});
