/*
 Client-side helper functions
 */

import {config} from './main'
import {escape} from '../vendor/underscore'

/**
 * Make spoiler tags toggleable on mobile
 * @param {Element}
 */
export function touchable_spoiler_tag(del) {
	del.html = '<del onclick="void(0)">'
}

/**
 * Retrieve post number of post element
 * @param {Element} el
 * @returns {int}
 */
export function getNum(el) {
	if (!el) {
	    return 0
	}
	return parseInt(el.getAttribute('id').slice(1), 10)
}

/**
 * Retrieve post number of closest parent post
 * @param {Element} el
 * @returns {int}
 */
export function getID(el) {
	if (!el) {
    	return 0
	}
	return getNum(el.closest('article, section'))
}

/**
 * Parse HTML string to node array
 * @param {string} string
 * @returns {Element[]}
 */
export function parseEls(string) {
	const el = document.createElement('div')
	el.innerHTML = string
	const children = el.childNodes
	return Array.from(children)
}

/**
 * Parse HTML string to Element
 * @param {string} string
 * @returns {Element}
 */
export function parseEl(string) {
	const el = document.createElement('div')
	el.innerHTML = string
	return el.firstChild
}

/**
 * Add an event listener that filters targets according to a CSS selector
 * @param {Element} el - Target element
 * @param {string} type - Event type
 * @param {string} selector - CSS selector
 * @param {function} handler - Callback function
 */
export function listener(el, type, selector, handler) {
	el.addEventListener(type, function (event) {
		if (event.target.matches(selector)) {
			handler.call(this, event)
		}
	})
}

/**
 * Add event listener to element, that will only be executed once
 * @param {Element} el - Target element
 * @param {string} type	- Event type
 * @param {function} handler - Callback function
 */
export function once(el, type, handler) {
	el.addEventListener(type, function (event) {
		handler.call(this, event)
		el.removeEventListener(type, handler)
	})
}

/**
 * Return width of element with padding and margin
 * @param {Element} el
 * @returns {int}
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

/**
 * Confirms email is saging
 * @param {string} email
 * @returns {bool}
 */
export function isSage(email) {
	return email && email.trim() === 'sage'
}

// TODO: Refactor server time syncronisation
let cachedOffset;
export function serverTime() {
	const d = Date.now();
	if (imports.isNode)
		return d;

	// The offset is intialised as 0, so there is something to return, until
	// we get a propper number from the server.
	if (!cachedOffset)
		cachedOffset = imports.main.request('time:offset');
	return d + cachedOffset;
}

export function readable_dice(bit, dice) {
	let inner;
	switch (bit) {
		case '#flip':
			inner = (dice[2] == 2).toString();
			break;
		case '#8ball':
			inner = imports.hotConfig.EIGHT_BALL[dice[2] - 1];
			break;
		case '#pyu':
		case '#pcount':
		case '#q':
			inner = dice[0];
			break;
	}
	if (inner !== undefined)
		return escape(`${bit} (${inner})`);
	if (/^#sw/.test(bit))
		return readableSyncwatch(dice[0]);
	return readableRegularDice(bit, dice);
}

function readableSyncwatch(dice) {
	dice.class = 'embed';
	return parseHTML
		`<syncwatch ${dice}>
			syncwatch
		</syncwatch>`;
}

function readableRegularDice(bit, dice) {
	const [max, bias, ...rolls] = dice
	bit += ' (';
	const eq = rolls.length > 1 || bias;
	if (eq)
		bit += rolls.join(', ');
	if (bias)
		bit += (bias < 0 ? ' - ' + (-bias) : ' + ' + bias);
	let sum = bias;
	for (let roll of rolls) {
		sum += roll;
	}
	return bit + (eq ? ' = ' : '') + sum + ')';
}

export function pick_spoiler(metaIndex) {
	const imgs = imports.config.SPOILER_IMAGES,
		n = imgs.length;
	let i;
	if (metaIndex < 0)
		i = Math.floor(Math.random() * n);
	else
		i = metaIndex % n;
	return {
		index: imgs[i],
		next: (i + 1) % n
	};
}

export const thumbStyles = ['small', 'sharp', 'hide']

export function readable_filesize(size) {
	/* Dealt with it. */
	if (size < 1024)
		return size + ' B';
	if (size < 1048576)
		return Math.round(size / 1024) + ' KB';
	size = Math.round(size / 104857.6).toString();
	return size.slice(0, -1) + '.' + size.slice(-1) + ' MB';
}

export function pad(n) {
	return (n < 10 ? '0' : '') + n;
}

// Various UI-related links wrapped in []
export function action_link_html(href, name, id, cls) {
	return parseHTML
		`<span class="act">
			<a href="${href}"
				${id && ` id="${id}"`}
				${cls && ` class="${cls}"`}
			>
				${name}
			</a>
		</span>`;
}

/**
 * Confirm last N posts to view setting matches bounds
 * @param {int} n
 * @returns {bool}
 */
export function resonableLastN(n) {
	return Number.isInteger(n) && n <= 500
}

export function parse_name(name) {
	var tripcode = '', secure = '';
	var hash = name.indexOf('#');
	if (hash >= 0) {
		tripcode = name.substr(hash + 1);
		name = name.substr(0, hash);
		hash = tripcode.indexOf('#');
		if (hash >= 0) {
			secure = escape(tripcode.substr(hash + 1));
			tripcode = tripcode.substr(0, hash);
		}
		tripcode = escape(tripcode);
	}
	name = name.trim().replace(imports.hotConfig.EXCLUDE_REGEXP, '');
	return [
		name.substr(0, 100), tripcode.substr(0, 128),
		secure.substr(0, 128)
	];
}

/**
 * Generate a random alphannumeric string of lower and upper case hexadecimal
 * characters
 * @param {int} len	- String length
 * @returns {string}
 */
export function randomID(len) {
	let id = ''
	for (let i = 0; i < len; i++) {
		let char = (Math.random() * 36).toString(36)[0]
		if (Math.random() < 0.5)
			char = char.toUpperCase()
		id += char
	}
	return id
}

/**
 * Template string tag function for HTML. Strips indentation and trailing
 * newlines. Based on https://gist.github.com/zenparsing/5dffde82d9acef19e43c
 * @param {*}
 * @returns {string}
 */
export function parseHTML(callSite) {
	// if arguments.length === 1
	if (typeof callSite === 'string') {
		return formatHTML(callSite);
	}

	/*
	 Slicing the arguments object is deoptimising, so we construct a new array
	 instead.
	 */
	const len = arguments.length
	const args = []
	for (let i = 1; i < len; i++) {
		args[i - 1] = arguments[i]
	}

	if (typeof callSite === 'function') {
		return formatHTML(callSite(args))
	}

	const output = callSite
		.slice(0, len)
		.map((text, i) =>
			args[i - 1] + text)
		.join('')

	return formatHTML(output)
}

/**
 * Strip indentation and remove empty lines from HTML string
 */
function formatHTML(str) {
	return str.replace(/\s*\n\s*/g, '')
}

/**
 * Generate an HTML element attribute list
 * @param {Object} attrs
 * @returns {string}
 */
export function parseAtributes(attrs) {
	let html = ''
	for (let key in attrs) {
		html += ' '
		const val = attrs[key]
		if (val === true)
			html += key
		else if (val || val === 0)
			html += `${key}="${val}"`
	}
	return html
}

/**
 * Makes a ', ' seperated list out of on array of strings
 * @param {string[]} items
 * @returns {string}
 */
export function commaList(items) {
	let html = ''
	for (let item of items) {
		// Falsy value. Skip item.
		if (!item && item !== 0)
			continue
		if (html)
			html += ', '
		html += item
	}
	return html
}

/**
 * Acertains client has the proper authorisation to perfrom task. This is only
 * for rendering. The same validation is performed server-side.
 * @param {string} action - Privilidged action to check permission for
 * @returns {bool}
 */
export function checkAuth(action) {
	const cls = config.staff.classes[main.ident && main.ident.auth]
    return cls && !!cls.rights[action]
}
