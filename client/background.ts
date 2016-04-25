// Background controller. Wallpapers, proper fitting and video backgrounds.

import stackBlur from './stackBlur'
import options from './options'
import {config, displayLoading} from './state'
import {parseHTML, load} from './util'
import {db} from './db'

const container = document.createElement('div')
const style = document.createElement('style')

container.id = 'user-background'
document.body.append(container)
document.head.append(style)

type BackgroundStore = {
	id: string
	normal: Blob
	blurred: Blob
}

// Central render function. Resets state and renders the apropriate background.
export function render(bg?: BackgroundStore) {
	container.innerHTML = ''
	style.innerHTML = ''
	if (options.get('illyaDance') && config.illyaDance) {
		renderIllya()
	} else if (options.get('userBG') && !options.get('workModeToggle')) {
		renderBackground(bg)
	}
}

// Listen to  changes in related options, that do not call render() directly
const changeProps = [
	'illyaDance', 'illyaDanceMute', 'theme', 'workModeToggle'
]
for (let param of changeProps) {
	options.onChange(param, render)
}

// Attach Illya Dance to the background
function renderIllya() {
	const urlBase = '/ass/illya.'
	let args = 'autoplay loop'
	if (options.get('illyaDanceMute')) {
		args += ' muted'
	}
	container.innerHTML = parseHTML
		`<video ${args}>
			<source src="${urlBase + 'webm'}" type="video/webm">
			<source src="${urlBase + 'mp4'}" type="video/mp4">
		</video>`
}

// Render a custom user-set background apply blurred glass to elements, if
// needed.
async function renderBackground(bg?: BackgroundStore) {
	if (!bg) {
		bg = await db
			.transaction(['main'], 'readonly')
			.objectStore('main')
			.get('background')
			.exec()
		if (!bg.normal || !bg.blurred) {
			return
		}
	}
	const normal = URL.createObjectURL(bg.normal)
	let html = parseHTML
		`#user-background {
			background: url(${normal}) no-repeat fixed center;
			background-size: cover;
		}`

	// Add blurred background image to elements, if theme is glass or ocean
	const theme = options.get('theme')
	if (theme === 'glass' || theme === 'ocean') {
		html += ' ' + renderGlass(theme, bg.blurred)
	}
	style.innerHTML = html
}

type BackgroundGradients = {
	normal: string
	editing: string
}

// Map for transparency gradients to apply on top of the blurred background
const colourMap: {[key: string]: BackgroundGradients} = {
	glass: {
		normal: 'rgba(40, 42, 46, 0.5)',
		editing: 'rgba(145, 145, 145, 0.5)'
	},
	ocean: {
		normal: 'rgba(28, 29, 34, 0.781)',
		editing: 'rgba(44, 57, 71, 0.88)'
	}
}

// Apply transparent blurred glass background to elemnts with the 'glass' class
function renderGlass(theme: string, blob: Blob): string {
	const {normal, editing} = colourMap[theme],
		blurred = URL.createObjectURL(blob)
	return parseHTML
		`.glass {
			background:
				linear-gradient(${normal}, ${normal}),
				url(${blurred}) center fixed no-repeat;
			background-size: cover;
		}
		.glass.editing, .editing .background {
			background:
				linear-gradient(${editing}, ${editing}),
				url(${blurred}) center fixed no-repeat;
			background-size: cover;
		}
		.background {
			padding: 10px;
			margin: 2px;
		}`
}

// Generate a blurred copy of the image and store both in IndexedDB. Then apply
// the new background, if enabled.
export async function store(file: File) {
	displayLoading(true)
	const img = new Image()
	img.src = URL.createObjectURL(file)
	await load(img)

	const canvas = document.createElement("canvas")
	canvas.width = img.width
	canvas.height = img.height
	canvas
		.getContext('2d')
		.drawImage(img, 0, 0, img.width, img.height)
	const normal = await canvasToBlob(canvas)

	// Generate blurred copy
	stackBlur(canvas, 0, 0, img.width, img.height, 10)
	const blurred = await canvasToBlob(canvas)

	const bg = {
		id: 'background',
		normal,
		blurred
	}

	await db
		.transaction(['main'], 'readwrite')
		.objectStore('main')
		.put(bg)
		.exec()

	if (options.get('userBG')) {
		render(bg)
	}
	displayLoading(false)
}

function canvasToBlob(canvas: HTMLCanvasElement): Promise<Blob> {
	return new Promise<Blob>(resolve => (canvas as any).toBlob(resolve))
}
