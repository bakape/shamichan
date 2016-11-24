// It is not very efficient to spam listeners to the options object. This
// module loops through the post models and calls the appropriate methods in
// batches.

import { posts, page } from "../state"
import options from "../options"
import { threads, write } from "../render"
import { fileTypes, Post } from "../posts/models"

// Listen for changes on the options object and call appropriate handlers on
// all applicable posts
const handlers: { [key: string]: () => void } = {
	workModeToggle: renderImages,
	hideThumbs: renderImages,
	spoilers: toggleSpoilers,
	autogif: toggleAutoGIF,
	anonymise: toggleAnonymisation,
	relativeTime: renderTime,
}
for (let key in handlers) {
	options.onChange(key, handlers[key])
}

// Rerender time every minute, if relative time is set
setInterval(renderTime, 60000)

// Loop over all posts after filtering with `test`
function loopPosts(test: (post: Post) => boolean, fn: (post: Post) => void) {
	for (let post of posts) {
		if (test(post)) {
			fn(post)
		}
	}
}

// Rerender all images
function renderImages() {
	if (!page.thread) {
		// Quick render, because we don't have models in the catalog
		let display = ""
		if (options.hideThumbs || options.workModeToggle) {
			display = "none"
		}
		write(() => {
			for (let el of threads.querySelectorAll(".expanded")) {
				el.style.display = display
			}
		})
	} else {
		loopPosts(
			({image}) =>
				!!image,
			({view}) =>
				view.renderImage(false, true),
		)
	}
}

// Image thumbnail spoilers
function toggleSpoilers() {
	loopPosts(
		({image}) =>
			!!image && image.spoiler,
		({view}) =>
			view.renderImage(false, true),
	)
}

// Animated GIF thumbnails
function toggleAutoGIF() {
	loopPosts(
		({image}) =>
			!!image && image.fileType === fileTypes.gif,
		({view}) =>
			view.renderImage(false, true),
	)
}

// Self-delusion tripfag filter
function toggleAnonymisation() {
	loopPosts(
		({name, trip, auth}) =>
			!!name || !!trip || !!auth,
		({view}) =>
			view.renderName(),
	)
}

// Rerender all timestamps on posts, if set to relative time
function renderTime() {
	if (!options.relativeTime) {
		return
	}
	for (let {view} of posts) {
		view.renderTime()
	}
}
