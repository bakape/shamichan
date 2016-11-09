import { Post, fileTypes } from "./models"
import View from "../view"
import { renderFigcaption, renderImage, sourcePath, } from "./render/image"
import { write, $threads } from "../render"
import options from "../options"
import { setAttrs, on } from "../util"
import { getModel, posts } from "../state"
import { trigger } from "../hooks"
import { images as lang } from "../lang"
import { deferInit } from "../defer"
import { scrollToElement } from "../scroll"

// Expand all image thumbnails automatically
export let expandAll = false

// Mixin for image expansion and related functionality
export default class ImageHandler extends View<Post> {
	// Render the figure and figcaption of a post. Set reveal to true, if in
	// hidden thumbnail mode, to reveal the thumbnail. Set delay to false to
	// only write the changes to DOM on the next animation frame.
	renderImage(reveal: boolean, delay: boolean) {
		const fn = () => {
			const img = this.model.image
			renderFigcaption(this.el.querySelector("figcaption"), img, reveal)
			renderImage(this.el.querySelector("figure"), img, reveal)
		}
		if (delay) {
			write(fn)
		} else {
			fn()
		}
	}

	toggleImageExpansion(event: Event) {
		const img = this.model.image
		if (img.expanded) {
			event.preventDefault()
			return this.contractImage(true, true)
		}

		switch (img.fileType) {
			case fileTypes.pdf:
				event.preventDefault()
				window.open(sourcePath(img.SHA1, img.fileType), "_blank")
				return
			case fileTypes.mp3:
				event.preventDefault()
				return this.renderAudio()
			case fileTypes.mp4:
			case fileTypes.ogg:
				if (!this.model.image.video) {
					event.preventDefault()
					return this.renderAudio()
				}
			default:
				return this.expandImage(event, false)
		}
	}

	// Automatically expand an image, if expandAll is set
	autoExpandImage() {
		if (expandAll && shouldAutoExpand(this.model)) {
			this.expandImage(null, true)
		}
	}

	// Contract an image and optionally omit scrolling to post and delay the
	// rendering of the change to the next animation frame.
	contractImage(scroll: boolean, delay: boolean) {
		const img = this.model.image

		switch (img.fileType) {
			case fileTypes.ogg:
			case fileTypes.mp3:
			case fileTypes.mp4:
			case fileTypes.webm:
				write(() => {
					const $v = this.el.querySelector("video")
					if ($v) {
						$v.remove()
					}
					const $a = this.el.querySelector("audio")
					if ($a) {
						$a.remove()
					}
					(this.el.querySelector("figure img") as HTMLElement)
						.hidden = false
				})
				break
		}

		this.renderImage(false, delay)

		// Scroll the post back into view, if contracting images taller than
		// the viewport
		if (img.tallerThanViewport && scroll) {
			scrollToElement(this.el)
		}

		img.expanded = img.tallerThanViewport = img.revealed = false
	}

	expandImage(event: Event | null, noScroll: boolean) {
		const mode = options.inlineFit,
			img = this.model.image
		let cls = "expanded "

		switch (mode) {
			case "none":
				return
			case "width":
				cls += "fit-to-width"
				img.tallerThanViewport = img.dims[1] > window.innerHeight
				if (img.tallerThanViewport && !noScroll) {
					scrollToElement(this.el)
				}
				break
			case "screen":
				cls += "fit-to-screen"
				break
		}
		this.model.image.expanded = true
		if (event) {
			event.preventDefault()
		}

		write(() => {
			// Hide any hover previews
			trigger("imageExpanded")

			const el = this.el.querySelector("figure img") as HTMLImageElement,
				src = sourcePath(img.SHA1, img.fileType)

			switch (img.fileType) {
				case fileTypes.ogg:
				case fileTypes.mp4:
				case fileTypes.webm:
					const video = document.createElement("video")
					setAttrs(video, {
						src,
						class: cls,
						autoplay: "",
						controls: "",
						loop: "",
					})
					el.hidden = true
					el.after(video)
					break
				default:
					setAttrs(el, {
						src,
						class: cls,
						width: "",
						height: "",
					})
			}
		})
	}

	// Render audio controls for uploaded MP3 files
	renderAudio() {
		const el = document.createElement("audio"),
			img = this.model.image
		setAttrs(el, {
			autoplay: "",
			loop: "",
			controls: "",
			src: sourcePath(img.SHA1, img.fileType),
		})
		this.model.image.expanded = true
		write(() =>
			this.el.querySelector("figure").after(el))
	}
}

// Delegate image clicks to views. More performant than dedicated listeners for
// each view.
function handleImageClick(event: MouseEvent) {
	if (options.inlineFit == "none" || event.which !== 1) {
		return
	}
	const model = getModel(event.target as Element)
	if (!model) {
		return
	}
	model.view.toggleImageExpansion(event)
}

// Reveal/hide thumbnail by clicking [Show]/[Hide] in hidden thumbnail mode
function toggleHiddenThumbnail(event: Event) {
	const model = getModel(event.target as Element)
	if (!model) {
		return
	}
	const {revealed} = model.image
	model.view.renderImage(!revealed, true)
	model.image.revealed = !revealed
}

// Toggle image expansion on [Expand Images] click
export function toggleExpandAll() {
	expandAll = !expandAll

	write(() => {
		const $e = $threads.querySelector("#expand-images")
		if ($e) {
			$e.textContent = expandAll ? lang.contract : lang.expand
		}
	})

	// Loop over all models and apply changes
	for (let post of posts) {
		if (!shouldAutoExpand(post)) {
			continue
		}
		if (expandAll) {
			post.view.expandImage(null, true)
		} else {
			post.view.contractImage(false, true)
		}
	}
}

// Resolve, if post should be automatically expanded or contracted
function shouldAutoExpand(model: Post): boolean {
	if (!model.image) {
		return false
	}
	switch (model.image.fileType) {
		case fileTypes.mp3:
		case fileTypes.mp4:
		case fileTypes.ogg:
		case fileTypes.pdf:
		case fileTypes.webm:
			return false
	}
	return true
}

deferInit(() => {
	on($threads, "click", handleImageClick, {
		selector: "img, video",
	})

	on($threads, "click", toggleHiddenThumbnail, {
		passive: true,
		selector: ".image-toggle",
	})

	on($threads, "click", toggleExpandAll, {
		passive: true,
		selector: "#expand-images",
	})
})
