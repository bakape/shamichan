// Background controller. Wallpapers, proper fitting and video backgrounds.

import stackBlur from './stackBlur'
import options from '.'
import { displayLoading } from '../state'
import { HTML, load } from '../util'
import { getObj, putObj } from '../db'

type BackgroundStore = {
	id: string
	normal: Blob
	blurred: Blob
}

type BackgroundGradients = {
	normal: string
	editing: string
	highlight: string
}

const container = document.getElementById("user-background"),
	style = document.getElementById("user-background-style")

// Map for transparency gradients to apply on top of the blurred background
const colourMap: { [key: string]: BackgroundGradients } = {
	glass: {
		normal: 'rgba(40, 42, 46, 0.5)',
		editing: 'rgba(105, 105, 105, 0.5)',
		highlight: 'rgba(64, 67, 73, .5)',
	},
	ocean: {
		normal: 'rgba(28, 29, 34, 0.78)',
		editing: 'rgba(44, 57, 71, 0.88)',
		highlight: 'rgba(44, 44, 51, 0.88)',
	}
}

// Listen to  changes in related options, that do not call render() directly
export default () => {
	for (let param of ['theme', 'workModeToggle']) {
		options.onChange(param, render)
	}
}

// Central render function. Resets state and renders the appropriate background.
export function render(bg?: BackgroundStore) {
	container.innerHTML = ''
	style.innerHTML = ''

	let showOPBG = false
	if (options.illyaDance) {
		renderIllya()
		showOPBG = true
	} else if (options.userBG && !options.workModeToggle) {
		renderBackground(bg)
		showOPBG = true
	}
	toggleOPBackground(showOPBG)
}

// Attach Illya Dance to the background
function renderIllya() {
	let args = 'autoplay loop'
	if (options.illyaDanceMute) {
		args += ' muted'
	}
	const html = HTML
		`<video ${args}>
			<source src="/assets/illya.webm" type="video/webm">
			<source src="/assets/illya.mp4" type="video/mp4">
		</video>`
	container.innerHTML = html
}

// Wrap the OP in a background for better visibility
function toggleOPBackground(on: boolean) {
	const tc = document.getElementById("thread-container")
	if (tc) {
		tc.classList.toggle("custom-BG", on)
	}
}

// Render a custom user-set background apply blurred glass to elements, if
// needed.
async function renderBackground(bg?: BackgroundStore): Promise<void> {
	if (!bg || !bg.normal || !bg.blurred) {
		bg = await getObj<BackgroundStore>("main", "background")
		if (!bg.normal || !bg.blurred) {
			return
		}
	}
	const normal = URL.createObjectURL(bg.normal)
	let html = HTML
		`#user-background {
			background: url(${normal}) no-repeat fixed center;
			background-size: cover;
		}`

	// Add blurred background image to elements, if theme is glass or ocean
	const {theme} = options
	if (theme === 'glass' || theme === 'ocean') {
		html += ' ' + renderGlass(theme, bg.blurred)
	}
	style.innerHTML = html
}

// Apply transparent blurred glass background to elements with the 'glass' class
function renderGlass(theme: string, blob: Blob): string {
	const {normal, editing, highlight} = colourMap[theme],
		blurred = URL.createObjectURL(blob)
	return HTML
		`.glass {
			background:
				linear-gradient(${normal}, ${normal}),
				url(${blurred}) center fixed no-repeat;
			background-size: cover;
		}
		article.editing {
			background:
				linear-gradient(${editing}, ${editing}),
				url(${blurred}) center fixed no-repeat;
			background-size: cover;
		}
		article.highlight:not(.editing), article:target {
			background:
				linear-gradient(${highlight}, ${highlight}),
				url(${blurred}) center fixed no-repeat;
			background-size: cover;
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
	await putObj("main", bg)

	if (options.userBG) {
		render(bg)
	}
	displayLoading(false)
}

function canvasToBlob(canvas: HTMLCanvasElement): Promise<Blob> {
	return new Promise<Blob>(resolve =>
		(canvas as any).toBlob(resolve))
}
