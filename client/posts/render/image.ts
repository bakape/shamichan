/*
 Image thumbnail HTML rendering
*/

import {config} from '../../state'
import options from '../../options'
import {
	parseHTML, commaList, parseAttributes, ElementAttributes, escape
} from '../../util'
import {ImageData, fileTypes} from '../models'
import {images as lang} from '../../lang'

// Render a thumbnail of an image, according to configuration settings
export function renderImage(data: ImageData, reveal?: boolean): string {
	const showThumb = options.hideThumbs || reveal
	return parseHTML
		`<figure>
			${renderFigcaption(data, reveal)}
			${config.images.hats && showThumb ? '<span class="hat"></span>': ''}
			${showThumb ? renderThumbnail(data) : ''}
		</figure>`
}

// Render the information caption above the image
export function renderFigcaption(data: ImageData, reveal: boolean): string {
	const list = commaList([
		data.audio ? '\u266B' : '',
		data.length.toString(),
		readableFilesize(data.size),
		`${data.dims[0]}x${data.dims[1]}`,
		data.apng ? 'APNG' : ''
	])
	return parseHTML
		`<figcaption>
			${hiddenToggle(reveal)}
			<span>
				(${list})
			</span>
			${imageLink(data)}
		</figcaption>`
}

// Renders a human readable file size string
function readableFilesize(size: number): string {
	if (size < 1024) {
		return size + ' B'
	}
	if (size < 1048576) {
		return Math.round(size / 1024) + ' KB'
	}
	const text = Math.round(size / 104857.6).toString()
	return `${text.slice(0, -1)}.${text.slice(-1)} MB`
}

// Render the button for toggling hidden thumbnails
function hiddenToggle(reveal: boolean): string {
	if (options.hideThumbs) {
		return ''
	}
	return parseHTML
		`<a class="imageToggle">
			[${lang[reveal ? 'hide' : 'show']}]
		</a>`
}

// Base URLs of image addresses
const imagePaths: {[type: string]: string} = {
	spoil: '/ass/spoil/spoiler'
}

// type ISTemplate = (data: ImageData) => string
//
// // Generate template functions for each image search engine
// const searchTemplates = (function() {
// 	const models = [
// 		{
// 			engine: 'google',
// 			url: 'https://www.google.com/searchbyimage?image_url=',
// 			type: 'thumb',
// 			symbol: 'G'
// 		},
// 		{
// 			engine: 'iqdb',
// 			url: 'http://iqdb.org/?url=',
// 			type: 'thumb',
// 			symbol: 'Iq'
// 		},
// 		{
// 			engine: 'saucenao',
// 			url: 'http://saucenao.com/search.php?db=999&url=',
// 			type: 'thumb',
// 			symbol: 'Sn'
// 		},
// 		{
// 			engine: 'desustorage',
// 			type: 'MD5',
// 			url: 'https://desustorage.org/_/search/image/',
// 			symbol: 'Ds'
// 		},
// 		{
// 			engine: 'exhentai',
// 			type: 'SHA1',
// 			url: 'http://exhentai.org/?fs_similar=1&fs_exp=1&f_shash=',
// 			symbol: 'Ex'
// 		}
// 	]
//
// 	const templates: {[engine: string]: ISTemplate} = {}
// 	for (let {engine, url, type, symbol} of models) {
// 		const attrs: ElementAttributes = {
// 			target: '_blank',
// 			rel: 'nofollow',
// 			class: 'imageSearch ' + engine
// 		}
// 		templates[engine] = data => {
// 			if (!options[engine]) {
// 				return ''
// 			}
// 			attrs['href'] = url
// 				+ (type === 'thumb' ? thumbPath(data, false) : data[type])
// 			return parseHTML
// 				`<a ${parseAttributes(attrs)}>
// 					${symbol}
// 				</a>`
// 		}
// 	}
//
// 	return templates
// })()
//
// // Render image search links in accordance to client settings
// function imageSearch(data: ImageData): string {
// 	let html = ''
//
// 	// Only render google for PDFs
// 	if (data.fileType === fileTypes.pdf) {
// 		if (options.google) {
// 			return searchTemplates['google'](data)
// 		}
// 		return ''
// 	}
// 	for (let engine in searchTemplates) {
// 		html += searchTemplates[engine](data)
// 	}
// 	return html
// }

// Get the thumbnail path of an image, accounting for not thumbnail of specific
// type being present
function thumbPath({SHA1, fileType}: ImageData): string {
	const ext = fileType === fileTypes.jpg ? "jpg" : "png"
	return `/img/thumb/${SHA1}.${ext}`
}

// Resolve the path to the source file of an upload
function sourcePath({SHA1, fileType}: ImageData): string {
	return `/img/src/${SHA1}.${fileTypes[fileType]}`
}


// Render a name + download link of an image
function imageLink(data: ImageData): string {
	let name = '',
		{file, fileType, imgnm} = data
	const m = imgnm.match(/^(.*)\.\w{3,4}$/)
	if (m) {
		name = m[1]
	}
	const fullName = escape(imgnm),
		tooLong = name.length >= 38

	if (tooLong) {
		imgnm = escape(name.slice(0, 30))
			+ '(&hellip;)'
			+ escape(fileTypes[fileType])
	}

	const attrs: ElementAttributes = {
		href: sourcePath(data),
		rel: 'nofollow',
		download: fullName
	}
	if (tooLong) {
		attrs['title'] = fullName
	}

	return parseHTML
		`<a ${parseAttributes(attrs)}>
			${imgnm}
		</a>`
}

// Render a hat on top of the thumbnail, if enabled
function renderHat(showThumb: boolean): string {
	if (showThumb && config.images.hats) {
		return '<span class="hat"></span>'
	}
	return ''
}

// Render the actual thumbnail image
export function renderThumbnail(data: ImageData, href?: string): string {
	const src = sourcePath(data)
	let thumb: string,
		[width, height, thumbWidth, thumbHeight] = data.dims

	if (data.spoiler && options.spoilers) {
		// Spoilered and spoilers enabled
		thumb = imagePaths['spoil'] + data.spoiler + '.jpg'
		thumbWidth = thumbHeight = 250
	} else if (data.fileType === fileTypes.gif && options.autogif) {
		// Animated GIF thumbnails
		thumb = src
	} else {
		thumb = thumbPath(data)
	}

	const linkAttrs: ElementAttributes = {
		target: '_blank',
		rel: 'nofollow',
		href: href || src
	}
	const imgAttrs: ElementAttributes = {
		src: thumb,
		width: thumbWidth.toString(),
		height: thumbHeight.toString()
	}

	// Catalog pages
	if (href) {
		// Handle the thumbnails with the HTML5 History controller
		linkAttrs['class'] = 'history'

		// No image hover previews
		imgAttrs['class'] = 'expanded'
		if (options.hideThumbs) {
			imgAttrs['style'] = 'display: none'
		}
	}

	return parseHTML
		`<a ${parseAttributes(linkAttrs)}>
			<img ${parseAttributes(imgAttrs)}>
		</a>`
}
