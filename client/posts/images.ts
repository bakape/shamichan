import { Post } from "./model"
import { fileTypes } from "../common"
import { View } from "../base"
import { renderImage, sourcePath } from "./render"
import { setAttrs, on, trigger } from "../util"
import options from "../options"
import { getModel, posts } from "../state"
import lang from "../lang"

// Expand all image thumbnails automatically
export let expandAll = false

// Mixin for image expansion and related functionality
export default class ImageHandler extends View<Post> {
	// Render the figure and figcaption of a post. Set reveal to true, if in
	// hidden thumbnail mode, to reveal the thumbnail.
	public renderImage(reveal: boolean) {
		renderImage(this.el, this.model.image, reveal)
	}

	public toggleImageExpansion(event: MouseEvent) {
		const img = this.model.image
		if (img.expanded) {
			return this.contractImage(event, true)
		}

		switch (img.fileType) {
			// Simply download the file
			case fileTypes.pdf:
			case fileTypes.zip:
			case fileTypes["7z"]:
			case fileTypes["tar.gz"]:
			case fileTypes["tar.xz"]:
				event.preventDefault()
				return this.el.querySelector("figcaption a[download]").click()
			case fileTypes.mp3:
				event.preventDefault()
				return this.renderAudio()
			case fileTypes.mp4:
			case fileTypes.ogg:
				if (!this.model.image.video) {
					event.preventDefault()
					return this.renderAudio()
				} else {
					return this.expandImage(event, false)
				}
			default:
				return this.expandImage(event, false)
		}
	}

	// Automatically expand an image, if expandAll is set
	public autoExpandImage() {
		if (expandAll && shouldAutoExpand(this.model)) {
			this.expandImage(null, true)
		}
	}

	// Contract an image and optionally omit scrolling to post and delay the
	// rendering of the change to the next animation frame.
	public contractImage(e: MouseEvent | null, scroll: boolean) {
		const img = this.model.image

		switch (img.fileType) {
			case fileTypes.ogg:
			case fileTypes.mp3:
			case fileTypes.mp4:
			case fileTypes.webm:
				// Firefox provides no way of detecting, if the controls where
				// clicked instead of the video. Estimate this by height.
				if (e) {
					const max = (e.target as HTMLElement).offsetHeight - 25
					if (e.offsetY > max) {
						return
					}
				}

				const v = this.el.querySelector("video")
				if (v) {
					v.remove()
				}
				const a = this.el.querySelector("audio")
				if (a) {
					a.remove()
				}
				this.el.querySelector("figure img").hidden = false
				break
		}

		if (e) {
			e.preventDefault()
		}
		this.renderImage(false)

		// Scroll the post back into view, if contracting images taller than
		// the viewport
		if (img.tallerThanViewport && scroll) {
			this.el.scrollIntoView()
		}

		img.expanded = img.tallerThanViewport = img.revealed = false
	}

	public expandImage(event: Event | null, noScroll: boolean) {
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
					this.el.scrollIntoView()
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

		// Hide any hover previews
		trigger("imageExpanded")

		const figure = this.el.querySelector("figure"),
			imgEl = figure.querySelector("img"),
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
				imgEl.hidden = true
				figure.append(video)
				break
			default:
				setAttrs(imgEl, {
					src,
					class: cls,
					width: "",
					height: "",
				})
		}
	}

	// Render audio controls for uploaded MP3 files
	private renderAudio() {
		const el = document.createElement("audio"),
			img = this.model.image
		setAttrs(el, {
			autoplay: "",
			loop: "",
			controls: "",
			src: sourcePath(img.SHA1, img.fileType),
		})
		this.model.image.expanded = true
		this.el.querySelector("figure").after(el)
	}
}

// Delegate image clicks to views. More performant than dedicated listeners for
// each view.
function handleImageClick(event: MouseEvent) {
	if (options.inlineFit === "none" || event.which !== 1) {
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
	model.view.renderImage(!revealed)
	model.image.revealed = !revealed
}

// Toggle image expansion on [Expand Images] click
export function toggleExpandAll() {
	expandAll = !expandAll

	const e = threads.querySelector("#expand-images a")
	if (e) {
		const k = (expandAll ? "contract" : "expand") + "Images"
		e.textContent = lang.posts[k]
	}

	// Loop over all models and apply changes
	for (let post of posts) {
		if (!shouldAutoExpand(post)) {
			continue
		}
		if (expandAll) {
			post.view.expandImage(null, true)
		} else {
			post.view.contractImage(null, false)
		}
	}
}

// Externally set the value of expandAll
export function setExpandAll(b: boolean) {
	expandAll = b
}

// Resolve, if post should be automatically expanded or contracted
function shouldAutoExpand(model: Post): boolean {
	if (!model.image) {
		return false
	}
	switch (model.image.fileType) {
		case fileTypes.jpg:
		case fileTypes.png:
		case fileTypes.gif:
			return true
		default:
			return false
	}
}

const threads = document.getElementById("threads")
on(threads, "click", handleImageClick, {
	selector: "img, video",
})
on(threads, "click", toggleHiddenThumbnail, {
	passive: true,
	selector: ".image-toggle",
})
on(threads, "click", toggleExpandAll, {
	passive: true,
	selector: "#expand-images a",
})
