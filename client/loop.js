/*
 It is not very efficient to spam liteners to the options object. This
 module loops through the post models and calls the appropriate methods in
 batch.
 */

const main = require('./main'),
	{etc, follow, options, oneeSama} = main,
	{posts} = main.state;

options.on({
	'change:thumbs': reRenderImages,
	'change:spoilers': toggleSpoilers,
	'change:autogif': toggleAutoGIF,
	'change:anonymise': toggleAnonymisation,
	'workModeTOG': reRenderImages
});
function reRenderImages() {
	if(main.state.page.get('catalog')){
		//quick render, because we don't have models in the catalog
		const show = (options.get("thumbs")!=='hide' && !main.oneeSama.workMode)? '':'none';
		document.queryAll(".expanded").forEach(el => el.style.display=show);
	}else
		follow(() => getImages((image, model) =>
			image && model.dispatch('renderImage', image)));
}

function getImages(func) {
	posts.each(model => func(model.get('image'), model));
}

function toggleSpoilers() {
	follow(() => getImages((image, model) =>
		image && image.spoiler && model.dispatch('renderImage', image)));
}

// Toggle animated GIF thumbnails
function toggleAutoGIF() {
	follow(() => getImages((image, model) =>
		image && image.ext === '.gif' && model.dispatch('renderImage', image)));
}

function toggleAnonymisation(source, toggle) {
	follow(() => {
		const command = toggle ? 'anonymise' : 'renderName';
		posts.each(function(model) {
			const {name, trip} = model.attributes;
			if (name || trip)
				model.dispatch(command);
		});
	});
}
main.reply('loop:anonymise', () =>
	options.get('anonymise') && toggleAnonymisation(null, true));
