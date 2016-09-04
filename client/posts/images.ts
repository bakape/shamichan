import {Post, fileTypes} from "./models"
import View from "../view"
import {renderFigcaption, renderImage, sourcePath} from "./render/image"
import {write, $threads} from "../render"
import options from "../options"
import {setAttrs, on} from "../util"
import {getModel} from "../state"
import {scrollToElement} from "../scroll"

// Mixin for image expansion and related functionality
export default class ImageHandler extends View<Post> {
	// Render the figure and figcaption of a post. Optionally set reveal to
	// true, if in hidden thumbnail mode, to reveal the thumbnail.
	renderImage(reveal?: boolean) {
		const img = this.model.image
		write(() =>
			(renderFigcaption(this.el.querySelector("figcaption"), img, reveal),
			renderImage(this.el.querySelector("figure"), img, reveal)))
	}

	toggleImageExpansion(event: Event) {
		const img = this.model.image
		if (img.expanded) {
			event.preventDefault()
			return this.contractImage()
		}

		switch (img.fileType) {
		case fileTypes.pdf:
			event.preventDefault()
			window.open(sourcePath(img.SHA1, img.fileType), "_blank")
			return
		case fileTypes.mp3:
			event.preventDefault()
			return this.renderAudio()
		default:
			return this.expandImage(event)
		}
	}

	contractImage() {
		const img = this.model.image
		if (img.length) {
			write(() =>
				this.el.querySelector("video").remove())
		}
		this.renderImage()

		// Scroll the post back into view, if contracting images taller than
		// the viewport
		if (img.tallerThanViewport) {
			scrollToElement(this.el as HTMLElement)
		}

		img.expanded = img.tallerThanViewport = img.revealed = false
	}

	expandImage(event: Event) {
		const mode = options.inlineFit,
			img = this.model.image
		let cls: string

		switch (mode) {
		case "none":
			return
		case "width":
			cls = "fit-to-width"
			img.tallerThanViewport = img.dims[1] > window.innerHeight
			break
		case "screen":
			cls = "fit-to-screen"
			break
		}
		this.model.image.expanded = true
		event.preventDefault()

		write(() => {
			const el = this.el.querySelector("figure img") as HTMLImageElement,
				src = sourcePath(img.SHA1, img.fileType)

			if (img.length) { // Only videos have a length property
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
			} else {
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
		// Audio controls are always the same height and do not need to be
		// fitted
		// TODO
	}
}

// Deleagte image clicks to views. More performant than dedicated listeners for
// each view.
on($threads, "click", handleImageClick, {
	selector: "img, video",
})

function handleImageClick(event: MouseEvent) {
	if (options.inlineFit == "none" || event.which !== 1) {
		return
	}
	const model = getModel(event.target as Element)
	if (!model) {
		return
	}
	model.view.toggleImageExpansion(event)

	// TODO: Remove any image hover previews
}

// Reveal/hide thumbnail by clicking [Show]/[Hide] in hidden thumbnail mode
on($threads, "click", toggleHiddenThumbnail, {
	passive: true,
	selector: ".image-toggle",
})

function toggleHiddenThumbnail(event: Event) {
	const model = getModel(event.target as Element)
	if (!model) {
		return
	}
	const {revealed} = model.image
	model.view.renderImage(!revealed)
	model.image.revealed = !revealed
}
