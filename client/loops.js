/*
 It is not very efficient to spam liteners to the options object. This
 module loops through the post models and calls the appropriate methods in
 batch.
 */

let main = require('./main'),
	{options, state} = main;

options.on({
	'change:thumbs': reRenderImages,
	'change:spoilers': toggleSpoilers,
	'change:autogif': toggleAutoGIF
});

function reRenderImages() {
	let models = state.posts.models;
	for (let i = 0, l = models.length; i < l; i++) {
		let model = models[i],
			image = model.get('image');
		if (image)
			model.dispatch('renderImage', image)
	}
}
main.comply('loop:images', () => reRenderImages());

function toggleSpoilers() {
	let models = state.posts.models;
	for (let i = 0, l = models.length; i < l; i++) {
		let model = models[i],
			image = model.get('image');
		if (image && image.spoiler)
			model.dispatch('renderImage', image);
	}
}

// Toggle animated GIF thumbnails
function toggleAutoGIF() {
	let models = state.posts.models;
	for (let i = 0, l = models.length; i < l; i++) {
		let model = models[i],
			image = model.get('image');
		if (image && image.ext === '.gif')
			model.dispatch('renderImage', image);
	}
}
