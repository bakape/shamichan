/*
 Client-side helper functions
 */

import {config} from './main'
import {escape} from 'underscore'

// Make spoiler tags toggleable on mobile
export function touchable_spoiler_tag(del) {
	del.innerHTML = '<del onclick="void(0)">'
}

// Retrieve post number of post element
export function getNum(el) {
	if (!el) {
		return 0
	}
	return parseInt(el.getAttribute('id').slice(1), 10)
}

// Retrieve post number of closest parent post
export function getID(el) {
	if (!el) {
		return 0
	}
	return getNum(el.closest('article, section'))
}


// Parse HTML string to node array
export function parseEls(string) {
	const el = document.createElement('div')
	el.innerHTML = string
	const children = el.childNodes
	return Array.from(children)
}

// Parse HTML string to single Element
export function parseEl(string) {
	const el = document.createElement('div')
	el.innerHTML = string
	return el.firstChild
}

// Add an event listener that filters targets according to a CSS selector
export function listener(el, type, selector, handler) {
	el.addEventListener(type, event => {
		if (event.target.matches(selector)) {
			handler(event)
		}
	})
}

// Add event listener to element, that will only be executed once
export function once(el, type, handler) {
	el.addEventListener(type, event => {
		handler(event)
		el.removeEventListener(type, handler)
	})
}

// Return width of element with padding and margin
export function outerWidth(el) {
	const style =  getComputedStyle(el),
		props = ['marginLeft', 'marginRight', 'paddingLeft','paddingRight']
	let width = 0
	for (let prop of props) {
		width += parseInt(style[prop]);
	}
	return width
}


// Confirms email is saging
export function isSage(email) {
	if (email) {
		return email.trim() === 'sage'
	}
	return false
}

// TODO: Refactor server time syncronisation
// let cachedOffset;
// export function serverTime() :number {
// 	const d = Date.now();
// 	if (imports.isNode)
// 		return d;
//
// 	// The offset is intialised as 0, so there is something to return, until
// 	// we get a propper number from the server.
// 	if (!cachedOffset)
// 		cachedOffset = imports.main.request('time:offset');
// 	return d + cachedOffset;
// }

// Pick the next spoiler from one of the available spoilers
export function pick_spoiler(metaIndex) {
	const imgs = config.SPOILER_IMAGES,
		n = imgs.length
	let i
	if (metaIndex < 0) {
		i = Math.floor(Math.random() * n)
	} else {
		i = metaIndex % n
	}
	return {
		index: imgs[i],
		next: (i + 1) % n
	}
}

export const thumbStyles = ['small', 'sharp', 'hide']

// Pad an integer with a leading zero, if below 10
export function pad(n) {
	return (n < 10 ? '0' : '') + n
}

// Various UI-related links wrapped in []
// export function action_link_html(href , name, id, cls) {
// 	return parseHTML
// 		`<span class="act">
// 			<a href="${href}"
// 				${id && ` id="${id}"`}
// 				${cls && ` class="${cls}"`}
// 			>
// 				${name}
// 			</a>
// 		</span>`;
// }

// Confirm last N posts to view setting matches bounds
export function resonableLastN(n) {
	return Number.isInteger(n) && n <= 500
}

// export function parse_name(name) {
// 	var tripcode = '', secure = '';
// 	var hash = name.indexOf('#');
// 	if (hash >= 0) {
// 		tripcode = name.substr(hash + 1);
// 		name = name.substr(0, hash);
// 		hash = tripcode.indexOf('#');
// 		if (hash >= 0) {
// 			secure = escape(tripcode.substr(hash + 1));
// 			tripcode = tripcode.substr(0, hash);
// 		}
// 		tripcode = escape(tripcode);
// 	}
// 	name = name.trim().replace(imports.hotConfig.EXCLUDE_REGEXP, '');
// 	return [
// 		name.substr(0, 100), tripcode.substr(0, 128),
// 		secure.substr(0, 128)
// 	];
// }

// Template string tag function for HTML. Strips indentation and trailing
// newlines. Based on https://gist.github.com/zenparsing/5dffde82d9acef19e43c
export function parseHTML(callSite, ...args) {
	let output = callSite[0]
	for (let i = 1; i <= args.length; i++) {
		output += args[i - 1] + callSite[i]
	}

	// Strip indentation and remove empty lines from HTML string
	return output.replace(/\s*\n\s*/g, '')
}


// Generate an HTML element attribute list
export function parseAttributes(attrs) {
	let html = ''
	for (let key in attrs) {
		html += ' '
		const val = attrs[key]
		if (val === true) {
			html += key
		} else if (val || val === 0) {
			html += `${key}="${val}"`
		}
	}
	return html
}

// Makes a ', ' seperated list out of on array of strings
export function commaList(items) {
	let html = ''
	for (let item of items) {
		if (html) {
			html += ', '
		}
		html += item
	}
	return html
}

// Acertains client has the proper authorisation to perfrom task. This is only
// for rendering. The same validation is performed server-side.
export function checkAuth(action) {
	const cls = config.staff.classes[main.ident && main.ident.auth]
	return cls && !!cls.rights[action]
}
