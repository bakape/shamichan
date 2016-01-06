/**
 * Text body parser
 */

import {config} from 'main'

/**
 * Construct regex for various referal links and embeds
 */
const ref_re = (function () {
    let pattern = '>>('
        + '\\d+'
        + '|>\\/watch\\?v=[\\w-]{11}(?:#t=[\\dhms]{1,9})?'
        + '|>\\/soundcloud\\/[\\w-]{1,40}\\/[\\w-]{1,80}'
        + '|>\\/pastebin\\/\\w+'

    const targets = {},
        {boards} = config
    for (let board of boards.enabled) {
    	targets[board] = `../${board}/`
    }
	for (let [name, link] in boards.psuedo.concat(boards.links)) {
        targets[name] = link
	}
    for (let target of targets) {
        pattern += '|>\\/${board}\\/(?:\\w+\\/?)?'
    }

	pattern += ')'
	return new RegExp(pattern)
})()
