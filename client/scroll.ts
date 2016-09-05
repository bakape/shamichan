// Various page scrolling aids

const $banner = document.querySelector("#banner") as HTMLElement

// Scroll to an element in the DOM with compensation for banner height
export function scrollToElement(el: HTMLElement) {
	const pos =
		el.getBoundingClientRect().top
		+ window.scrollY
		- $banner.offsetHeight
	window.scrollTo(0, pos)
}

// Scroll to target anchor element, if any
export function scrollToAnchor() {
	if (!location.hash) {
		return
	}
	scrollToElement(document.querySelector(location.hash) as HTMLElement)
}
