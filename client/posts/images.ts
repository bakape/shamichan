import { Post } from "./model"
import { fileTypes } from "../common"
import { View } from "../base"
import {
	setAttrs, on, trigger, firstChild, importTemplate, escape, pad
} from "../util"
import options from "../options"
import { getModel, posts, config } from "../state"
import lang from "../lang"

// Specs for handling image search link clicks
type ImageSearchSpec = {
	type: ISType
	url: string
}

// Types of data requested by the search provider
const enum ISType { src, MD5, SHA1 }

const ISSpecs: ImageSearchSpec[] = [
	{
		type: ISType.src,
		url: "https://www.google.com/searchbyimage?image_url=",
	},
	{
		type: ISType.src,
		url: "http://iqdb.org/?url=",
	},
	{
		type: ISType.src,
		url: "http://saucenao.com/search.php?db=999&url=",
	},
	{
		type: ISType.src,
		url: "https://whatanime.ga/?url=",
	},
	{
		type: ISType.MD5,
		url: "https://desuarchive.org/_/search/image/",
	},
	{
		type: ISType.SHA1,
		url: "http://exhentai.org/?fs_similar=1&fs_exp=1&f_shash=",
	},
]

// Expand all image thumbnails automatically
export let expandAll = false

// Mixin for image expansion and related functionality
export default class ImageHandler extends View<Post> {
	// Render the figure and figcaption of a post. Set reveal to true, if in
	// hidden thumbnail mode, to reveal the thumbnail.
	public renderImage(reveal: boolean) {
		this.el.classList.add("media")

		let el = this.getFigure()
		if (!el) {
			el = importTemplate("figure").firstChild as HTMLElement
			this.el.querySelector(".post-container").prepend(el)
		}

		const showThumb = (!options.hideThumbs && !options.workModeToggle)
			|| reveal
		el.hidden = !showThumb
		if (showThumb) {
			(el.firstElementChild as HTMLElement).hidden = false
			this.renderThumbnail()
		}
		this.renderFigcaption(reveal)
	}

	// Need to find direct descendant, otherwise inlined posts might match
	private getFigure(): HTMLElement {
		return firstChild(this.el.querySelector(".post-container"), ch =>
			ch.tagName === "FIGURE")
	}

	// Need to find direct descendant, otherwise inlined posts might match
	private getFigcaption(): HTMLElement {
		return firstChild(this.el, ch =>
			ch.tagName === "FIGCAPTION")
	}

	public removeImage() {
		this.el.classList.remove("media")
		let el = this.getFigure()
		if (el) {
			el.remove()
		}
		el = this.getFigcaption()
		if (el) {
			el.remove()
		}
	}

	// Render the actual thumbnail image
	private renderThumbnail() {
		const el = this.el.querySelector("figure a"),
			data = this.model.image,
			src = sourcePath(data.SHA1, data.fileType)
		let thumb: string,
			[, , thumbWidth, thumbHeight] = data.dims

		if (data.spoiler && options.spoilers) {
			// Spoilered and spoilers enabled
			thumb = '/assets/spoil/default.jpg'
			thumbHeight = thumbWidth = 150
		} else if (data.fileType === fileTypes.gif && options.autogif) {
			// Animated GIF thumbnails
			thumb = src
		} else {
			thumb = thumbPath(data.SHA1, data.thumbType)
		}

		// Downscale thumbnail for higher DPI, unless specified not to
		if (!data.large && (thumbWidth > 125 || thumbHeight > 125)) {
			thumbWidth *= 0.8333
			thumbHeight *= 0.8333
		}

		el.setAttribute("href", src)
		setAttrs(el.firstElementChild, {
			src: thumb,
			width: thumbWidth.toString(),
			height: thumbHeight.toString(),
			class: "", // Remove any existing classes
		})
	}

	// Render the information caption above the image
	private renderFigcaption(reveal: boolean) {
		let el = this.getFigcaption()
		if (!el) {
			el = importTemplate("figcaption").firstChild as HTMLElement
			this.el.querySelector("header").after(el)
		}

		const [hToggle, , info, link] = Array.from(el.children) as HTMLElement[]
		if (!options.hideThumbs && !options.workModeToggle) {
			hToggle.hidden = true
		} else {
			hToggle.hidden = false
			hToggle.textContent = lang.posts[reveal ? 'hide' : 'show']
		}

		const data = this.model.image
		for (let el of Array.from(info.children) as HTMLElement[]) {
			switch (el.className) {
				case "media-title":
					el.textContent = data.title
					break;
				case "media-artist":
					el.textContent = data.artist
					break
				case "has-audio":
					el.hidden = !data.audio
					break
				case "media-length":
					const len = data.length
					if (len) {
						let s: string
						if (len < 60) {
							s = `0:${pad(len)}`
						} else {
							const min = Math.floor(len / 60),
								sec = len - min * 60
							s = `${pad(min)}:${pad(sec)}`
						}
						el.textContent = s
					}
					break
				case "is-apng":
					el.hidden = !data.apng
					break
				case "filesize":
					const { size } = data
					let s: string
					if (size < (1 << 10)) {
						s = size + ' B'
					} else if (size < (1 << 20)) {
						s = Math.round(size / (1 << 10)) + ' KB'
					} else {
						const text = Math.round(size / (1 << 20) * 10)
							.toString()
						s = `${text.slice(0, -1)}.${text.slice(-1)} MB`
					}
					el.textContent = s
					break
				case "dims":
					el.textContent = `${data.dims[0]}x${data.dims[1]}`
					break
			}
		}

		// Render a name + download link of an image
		const ext = fileTypes[data.fileType],
			name = `${escape(data.name)}.${ext}`
		setAttrs(el.lastElementChild, {
			href: `/images/src/${data.SHA1}.${ext}`,
			download: name,
		})
		link.innerHTML = name

		// Assign URLs to image search links
		const ch = el.querySelector(".image-search-container").children
		for (let i = 0; i < ch.length; i++) {
			const { type, url } = ISSpecs[i]
			let arg: string
			switch (type) {
				case ISType.src:
					arg = this.resolveFuzzyIS()
					break
				case ISType.MD5:
					arg = data.MD5
					break
				case ISType.SHA1:
					arg = data.SHA1
					break
			}
			ch[i].setAttribute("href", url + arg)
		}

		el.hidden = false
	}

	// Resolve URL of image search providers, that require to download the
	// image file
	private resolveFuzzyIS(): string {
		const { fileType, thumbType, SHA1, size } = this.model.image
		let root: string,
			type: fileTypes
		switch (fileType) {
			case fileTypes.jpg:
			case fileTypes.gif:
			case fileTypes.png:
				if (size > 8 << 20) {
					root = "thumb"
					type = thumbType
				} else {
					root = "src"
					type = fileType
				}
				break
			default:
				root = "thumb"
				type = thumbType
		}
		const s = `/images/${root}/${SHA1}.${fileTypes[type]}`
		return encodeURI(location.origin + s)
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
				const el = document.createElement("img")
				setAttrs(el, {
					src,
					class: cls,
				})
				imgEl.replaceWith(el)
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

function imageRoot(): string {
	return config.imageRootOverride || "/images"
}

// Get the thumbnail path of an image, accounting for not thumbnail of specific
// type being present
export function thumbPath(SHA1: string, thumbType: fileTypes): string {
	return `${imageRoot()}/thumb/${SHA1}.${fileTypes[thumbType]}`
}

// Resolve the path to the source file of an upload
export function sourcePath(SHA1: string, fileType: fileTypes): string {
	return `${imageRoot()}/src/${SHA1}.${fileTypes[fileType]}`
}

// Delegate image clicks to views. More performant than dedicated listeners for
// each view.
function handleImageClick(event: MouseEvent) {
	const el = event.target as Element
	const bypass = options.inlineFit === "none"
		|| event.which !== 1
		|| el.classList.contains("catalog")
	if (bypass) {
		return
	}
	const model = getModel(el)
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
	const { revealed } = model.image
	model.view.renderImage(!revealed)
	model.image.revealed = !revealed
}

// Toggle image expansion on [Expand Images] click
export function toggleExpandAll() {
	expandAll = !expandAll

	const e = document.querySelector("#expand-images a")
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

on(document, "click", handleImageClick, {
	selector: "img, video",
})
on(document, "click", toggleHiddenThumbnail, {
	passive: true,
	selector: ".image-toggle",
})
on(document, "click", toggleExpandAll, {
	passive: true,
	selector: "#expand-images a",
})
