/*
 Image thumbnail HTML rendering
*/

import lang from 'lang'
import {config} from '../../state'
import {escape} from 'underscore'
import options from '../../options'
import {parseHTML, commaList} from '../../util'

// Render a thumbnail of an image, according to configuration settings
export function renderImage(data, reveal) {
	const showThumb = options.get("thumbs") !== 'hide' || reveal
	return parseHTML
		`<figure>
			${renderFigcaption(data, reveal)}
			${config.images.hat && showThumb ? '<span class="hat"></span>': ''}
			${showThumb ? renderThumbnail(data) : ''}
		</figure>`
}

// Render the information caption above the image
export function renderFigcaption(data, reveal) {
	const list = commaList([
		data.audio ? '\u266B' : '',
		data.length,
		readableFilesize(data.size),
		`${data.dims[0]}x${data.dims[1]}`,
		data.apng ? 'APNG' : ''
	])
	return parseHTML
		`<figcaption>
			${hiddenToggle(reveal)}
			${imageSearch(data)}
			<span>
				(${list})
			</span>
			${imageLink(data)}
		</figcaption>`
}

// Renders a human readable file size string
function readableFilesize(size) {
	if (size < 1024) {
		return size + ' B'
	}
	if (size < 1048576) {
		return Math.round(size / 1024) + ' KB'
	}
	size = Math.round(size / 104857.6).toString()
	return size.slice(0, -1) + '.' + size.slice(-1) + ' MB'
}

// Render the button for toggling hidden thumbnails
function hiddenToggle(reveal) {
	if (options.get('thumbs') !== 'hide') {
		return ''
	}
	return parseHTML
		`<a class="imageToggle">
			[${lang[reveal ? 'hide' : 'show']}]
		</a>`
}

// Base URLs of image addresses
const imagePaths = {
	src: '/img/src/',
	thumb: '/img/thumb/',
	mid: '/img/mid/',
	spoil: '/ass/spoil/spoiler'
}

// Generate template functions for each image search engine
const searchTemplates = (function() {
	const models = [
		{
			engine: 'google',
			url: 'https://www.google.com/searchbyimage?image_url=',
			type: 'thumb',
			symbol: 'G'
		},
		{
			engine: 'iqdb',
			url: 'http://iqdb.org/?url=',
			type: 'thumb',
			symbol: 'Iq'
		},
		{
			engine: 'saucenao',
			url: 'http://saucenao.com/search.php?db=999&url=',
			type: 'thumb',
			symbol: 'Sn'
		},
		{
			engine: 'desustorage',
			type: 'MD5',
			url: 'https://desustorage.org/_/search/image/',
			symbol: 'Ds'
		},
		{
			engine: 'exhentai',
			type: 'SHA1',
			url: 'http://exhentai.org/?fs_similar=1&fs_exp=1&f_shash=',
			symbol: 'Ex'
		}
	]

	const templates = {}
	for (let {engine, url, type, symbol} of models) {
		const attrs = {
			target: '_blank',
			rel: 'nofollow',
			class: 'imageSearch ' + engine
		}
		templates[engine] = data => {
			if (!options.get(engine)) {
				return ''
			}
			attrs.href = url+ (type === 'thumb' ? thumbPath(data) : data[type])
			return parseHTML
				`<a ${parseAttributes(attrs)}>
					${symbol}
				</a>`
		}
	}

	return templates
})()

// Render image search links in accordance to client settings
function imageSearch(data) {
	let html = ''

	// Only render google for PDFs and MP3s
	if (['.pdf', '.mp3'].indexOf(data.ext) > -1) {
		if (options.get("google")) {
			return searchTemplates.google(data)
		}
		return ''
	}
	for (let engine in searchTemplates) {
		html += searchTemplates[engine](data)
	}
	return html
}

// Get the thumbnail path of an image, accounting for not thumbnail of specific
// type being present
function thumbPath(data, mid) {
	return imagePaths[type] + data[mid && data.mid ? 'mid' : 'thumb']
}

// Render a name + download link of an image
function imageLink(data) {
	let name = '',
		{imgnm} = imgnm
	const m = imgnm.match(/^(.*)\.\w{3,4}$/);
	if (m) {
		name = m[1]
	}
	const fullName = escape(imgnm),
		tooLong = name.length >= 38
	if (tooLong) {
		imgnm = escape(name.slice(0, 30))
			+ '(&hellip;)'
			+ escape(data.ext)
	}
	const attrs = {
		href: `${config.SECONDARY_MEDIA_URL}src/${data.src}`,
		rel: 'nofollow',
		download: fullName
	}
	if (tooLong) {
		attrs.title = fullName
	}
	return parseHTML
		`<a ${parseAttributes(attrs)}>
			${imgnm}
		</a>`
}

// Render a hat on top of the thumbnail, if enabled
function renderHat(showThumb) {
	if (showThumb && config.images.hats) {
		return '<span class="hat"></span>'
	}
	return ''
}

// Render the actual thumbnail image
export function renderThumbnail(data, href) {
	let src = imagePaths.src + data.src,
		thumb,
		[width, height, thumbWidth, thumbHeight] = data.dims

	if (data.spoiler && options.get('spoilers')) {
		// Spoilered and spoilers enabled
		const sp = spoilerInfo(data)
		thumb = sp.thumb
		thumbWidth = sp.dims[0]
		thumbHeight = sp.dims[1]
	} else if (data.ext === '.gif' && options.get('autogif')) {
		// Animated GIF thumbnails
		thumb = src
	} else {
		thumb = thumbPath(data, options.get('thumbs') !== 'small')
	}

	const linkAttrs = {
		target: '_blank',
		rel: 'nofollow',
		href: href || src
	}
	const imgAttrs = {
		src: thumb,
		width: thumbWidth,
		height: thumbHeight
	}

	// Catalog pages
	if (href) {
		// Handle the thumbnails with the HTML5 History controller
		linkAttrs.class = 'history'

		// No image hover previews
		imgAttrs.class = 'expanded'
		if(this.thumbStyle == 'hide') {
			imgAttrs.style = 'display: none'
		}
	}

	return parseHTML
		`<a ${parseAttributes(linkAttrs)}>
			<img ${parseAttributes(imgAttrs)}>
		</a>`
}

// Parse and return image spoiler information
function spoilerInfo({largeThumb, spoiler}) {
	let thumb = imagePaths.spoil
	if (largeThumb || options.get("thumbs") !== 'small') {
		thumb += 's'
	}
	html += spoiler + '.png'
	return {
		thumb,
		dims: config.images.thumb[largeThumb ? 'midDims' : 'thumbDims']
	}
}
