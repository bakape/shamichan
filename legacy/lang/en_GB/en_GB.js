/*
 Client-side language pack
*/

// TODO: Port moderation to the new client
// Moderation
export const mod = {
	title: ['Title', 'Display staff title on new posts'],
	clearSelection: ['Clear', 'Clear selected posts'],
	spoilerImages: ['Spoiler', 'Spoiler selected post images'],
	deleteImages: ['Del Img', 'Delete selected post images'],
	deletePosts: ['Del Post', 'Delete selected posts'],
	lockThreads: ['Lock', 'Lock/unlock selected threads'],
	toggleMnemonics: ['Mnemonics', 'Toggle mnemonic display'],
	sendNotification: [
		'Notification',
		'Send notifaction message to all clients'
	],
	ban: ['Ban', 'Ban poster(s) for the selected post(s)'],
	renderPanel: ['Panel', 'Toggle administrator panel display'],
	modLog: ['Log', 'Show moderation log'],
	djPanel: ['DJ', 'DJ tool panel'],
	displayBan: [
		'Display',
		'Append a public \'USER WAS BANNED FOR THIS POST\' message'
	],
	unban: 'Unban',
	banMessage: 'USER WAS BANNED FOR THIS POST',
	placeholders: {
		msg: 'Message',
		days: 'd',
		hours: 'h',
		minutes: 'min',
		reason: 'Reason'
	},
	needReason: 'Must specify reason',

	// Correspond to websocket calls in common/index.js
	7: 'Image spoilered',
	8: 'Image deleted',
	9: 'Post deleted',
	10: 'Thread locked',
	11: 'Thread unlocked',
	12: 'User banned',
	53: 'User unbanned',
}

// Format the moderation entries visible to all staff
export function formatModeration(act) {
	var msg = lang.mod[act.kind] + ' by ' + act.ident
	if (act.reason) {
		msg += ' for ' + act.reason
	}
	return msg
}

// 56 minutes ago / in 56 minutes
export function ago(time, unit, isFuture) {
	var res = pluralize(time, unit)
	if (isFuture) {
		res = 'in ' + res
	} else {
		res += ' ago'
	}
	return res
}

// 47 replies and 21 images omitted
export function abbrevMsg(omit, img_omit, url) {
	var html = pluralize(omit, 'reply')
	if (img_omit) {
		html += ' and ' + pluralize(img_omit, 'image')
	}
	html += ' omitted'
	if (url) {
		html += ` <span class="act"><a href="${url}" class="history">`
			+ lang.see_all + '</a></span>'
	}
	return html
}
