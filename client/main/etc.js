/*
 Client-side helper functions
 */

import {_, config, state} from 'main'

/**
 * Make spoiler tags toggleable on mobile
 */
export function touchable_spoiler_tag(del) {
	del.html = '<del onclick="void(0)">';
}

/**
 * Return image upload URL
 */
export function imageUploadURL() {
	return (config.hard.HTTP.upload || '../upload/') + '?id='
		+ state.page.get('connID')
}

/**
 * Retrieve post number of post element
 */
export function getNum(el) {
	return el && parseInt(el.getAttribute('id').slice(1), 10)
}

/**
 * Retrieve post number of closest parent post
 */
export function getID(el) {
	return el && getNum(el.closest('article, section'))
}

/**
 * Retrieve model of closest parent post
 */
export function getModel(el) {
	const id = getID(el)
	if (!id)
		return null;
	return state.posts.get(id)
}

/**
 * Parse HTML string to node array/element
 */
export function parseDOM(string, forceArray) {
	const el = document.createElement('div')
	el.innerHTML = string
	const children = el.childNodes
	if (!forceArray && children.length === 1)
		return children[0]
	return Array.from(children)
}

/**
 * Create array of elements or single element from template string
 */
export function parseEls(string, forceArray) {
    return parseDOM(main.common.parseHTML(string), forceArray)
}

/**
 * Add an event listener that filters targets according to a CSS selector
 */
export function listener(el, type, selector, handler) {
	el.addEventListener(type, function (event) {
		if (event.target.matches(selector))
			handler.call(this, event)
	})
}

/**
 * Add event listener to element, that will only be executed once
 */
export function once(el, type, handler) {
	el.addEventListener(type, function (event) {
		handler.call(this, event)
		el.removeEventListener(type, handler)
	})
}

/**
 * Return width of element with padding and margin
 */
export function outerWidth(el) {
	const style =  getComputedStyle(el),
		props = ['marginLeft', 'marginRight', 'paddingLeft','paddingRight']
	let width = 0
	for (let prop of props) {
		width += parseInt(style[prop]);
	}
	return width
}
