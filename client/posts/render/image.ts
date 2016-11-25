import { boardConfig } from '../../state'
import options from '../../options'
import { commaList, escape, setAttrs, pad } from '../../util'
import { ImageData, fileTypes } from '../models'
import { images as lang } from '../../lang'

// Specs for handling image search link clicks
type ImageSearchSpec = {
	type: ISType
	url: string
}

// Types of data requested by the search provider
const enum ISType { thumb, MD5, SHA1 }

const ISSpecs: ImageSearchSpec[] = [
	{
		type: ISType.thumb,
		url: "https://www.google.com/searchbyimage?image_url=",
	},
	{
		type: ISType.thumb,
		url: "http://iqdb.org/?url=",
	},
	{
		type: ISType.thumb,
		url: "http://saucenao.com/search.php?db=999&url=",
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

// Render a thumbnail of an image, according to configuration settings
export function renderImage(
	el: HTMLElement,
	data: ImageData,
	reveal: boolean,
) {
	const showThumb = (!options.hideThumbs && !options.workModeToggle) || reveal
	el.hidden = !showThumb
	if (showThumb) {
		(el.firstElementChild as HTMLElement).hidden = false
	}
	if (showThumb) {
		renderThumbnail(el.lastElementChild, data)
	}
}

// Render the information caption above the image
export function renderFigcaption(
	el: HTMLElement,
	data: ImageData,
	reveal: boolean,
) {
	const list: string[] = []
	if (data.audio) {
		list.push('\u266B')
	}
	if (data.length) {
		list.push(readableLength(data.length))
	}
	if (data.apng) {
		list.push('APNG')
	}
	list.push(readableFilesize(data.size), `${data.dims[0]}x${data.dims[1]}`)

	const [hToggle, , info, link] = Array.from(el.children) as HTMLElement[]
	if (!options.hideThumbs && !options.workModeToggle) {
		hToggle.hidden = true
	} else {
		hToggle.hidden = false
		hToggle.textContent = lang[reveal ? 'hide' : 'show']
	}
	info.textContent = `(${commaList(list)})`
	imageLink(link, data)
	renderImageSearch(el.querySelector(".image-search-container"), data)
	el.hidden = false
}

// Assign URLs to image search links
function renderImageSearch(cont: HTMLElement, img: ImageData) {
	const ch = cont.children
	for (let i = 0; i < ch.length; i++) {
		const {type, url} = ISSpecs[i]
		let arg: string
		switch (type) {
			case ISType.thumb:
				arg = location.origin + thumbPath(img.SHA1, img.fileType)
				break
			case ISType.MD5:
				arg = img.MD5
				break
			case ISType.SHA1:
				arg = img.SHA1
				break
		}
		ch[i].setAttribute("href", url + encodeURIComponent(arg))
	}
}

// Render video/audio length in human readable form
function readableLength(len: number): string {
	if (len < 60) {
		return `0:${pad(len)}`
	}
	const min = Math.floor(len / 60),
		sec = len - min * 60
	return `${pad(min)}:${pad(sec)}`
}

// Renders a human readable file size string
function readableFilesize(size: number): string {
	if (size < (1 << 10)) {
		return size + ' B'
	}
	if (size < (1 << 20)) {
		return Math.round(size / (1 << 10)) + ' KB'
	}
	const text = Math.round(size / (1 << 20) * 10).toString()
	return `${text.slice(0, -1)}.${text.slice(-1)} MB`
}

// Get the thumbnail path of an image, accounting for not thumbnail of specific
// type being present
export function thumbPath(SHA1: string, fileType: fileTypes): string {
	const ext = fileType === fileTypes.jpg ? "jpg" : "png"
	return `/images/thumb/${SHA1}.${ext}`
}

// Resolve the path to the source file of an upload
export function sourcePath(SHA1: string, fileType: fileTypes): string {
	return `/images/src/${SHA1}.${fileTypes[fileType]}`
}

// Render a name + download link of an image
export function imageLink(el: Element, data: ImageData) {
	let {name} = data
	const {fileType} = data,
		ext = fileTypes[fileType],
		fullName = `${escape(name)}.${ext}`,
		tooLong = name.length >= 38
	const attrs: { [key: string]: string } = {
		href: sourcePath(data.SHA1, data.fileType),
		download: fullName,
	}

	if (tooLong) {
		name = `${escape(name.slice(0, 30))}(&hellip;).${ext}`
		attrs['title'] = fullName
	} else {
		name = fullName
	}

	setAttrs(el, attrs)
	el.innerHTML = name
}

// Render the actual thumbnail image
export function renderThumbnail(el: Element, data: ImageData) {
	const src = sourcePath(data.SHA1, data.fileType)
	let thumb: string

	if (data.spoiler && options.spoilers) {
		// Spoilered and spoilers enabled
		thumb = '/assets/spoil/' + boardConfig.spoiler
	} else if (data.fileType === fileTypes.gif && options.autogif) {
		// Animated GIF thumbnails
		thumb = src
	} else {
		thumb = thumbPath(data.SHA1, data.fileType)
	}

	el.setAttribute("href", src)
	setAttrs(el.firstElementChild, {
		src: thumb,
		class: "", // Remove any existing classes
	})
}
