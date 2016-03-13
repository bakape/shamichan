/*
 Client-side helper functions
 */

import {config} from './main'
import {escape} from 'underscore'

// Make spoiler tags toggleable on mobile
export function touchable_spoiler_tag(del) {
	del.innerHTML = '<del onclick="void(0)">'
}

// TODO: Refactor server time syncronisation
let cachedOffset;
export function serverTime() :number {
	const d = Date.now();
	if (imports.isNode)
		return d;

	// The offset is intialised as 0, so there is something to return, until
	// we get a propper number from the server.
	if (!cachedOffset)
		cachedOffset = imports.main.request('time:offset');
	return d + cachedOffset;
}

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

// Various UI-related links wrapped in []
export function action_link_html(href , name, id, cls) {
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

// Acertains client has the proper authorisation to perfrom task. This is only
// for rendering. The same validation is performed server-side.
export function checkAuth(action) {
	const cls = config.staff.classes[main.ident && main.ident.auth]
	return cls && !!cls.rights[action]
}
