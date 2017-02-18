// It is not very efficient to spam listeners to the options object. This
// module loops through the post models and calls the appropriate methods in
// batches.

import { posts, page } from "../state"
import options from "."
import { threads } from "../util"
import { Post } from "../posts"
import { fileTypes } from "../common"

// Listen for changes on the options object and call appropriate handlers on
// all applicable posts
export default () => {
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
}

// Rerender time every minute, if relative time is set
setInterval(() => {
	if (options.relativeTime && !page.catalog) {
		renderTime()
	}
}, 60000)

// Loop over all posts after filtering with `test`
function loopPosts(test: (post: Post) => boolean, fn: (post: Post) => void) {
	// Catalog does not contain proper views and is very static
	if (page.catalog) {
		return
	}
	for (let post of posts) {
		if (test(post)) {
			fn(post)
		}
	}
}

// Rerender all images
function renderImages() {
	if (page.catalog) {
		// Quick render, because we don't have views in the catalog
		let display = ""
		if (options.hideThumbs || options.workModeToggle) {
			display = "none"
		}
		for (let el of threads.querySelectorAll("img.catalog")) {
			el.style.display = display
		}
	} else {
		loopPosts(
			({image}) =>
				!!image,
			({view}) =>
				view.renderImage(false),
		)
	}
}

// Image thumbnail spoilers
function toggleSpoilers() {
	loopPosts(
		({image}) =>
			!!image && image.spoiler,
		({view}) =>
			view.renderImage(false),
	)
}

// Animated GIF thumbnails
function toggleAutoGIF() {
	loopPosts(
		({image}) =>
			!!image && image.fileType === fileTypes.gif,
		({view}) =>
			view.renderImage(false),
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
	for (let {view} of posts) {
		view.renderTime()
	}
}
