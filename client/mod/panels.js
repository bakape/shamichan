/*
Moderation log modal view
 */

const main = require('main'),
	{_, Backbone, common, dispatcher, etc, oneeSama, lang, modals, Memory,
		state} = main

const PanelView = Backbone.View.extend({
	className: 'modal mod panel',
	initialize() {
		this.$el.appendTo(main.$overlay);
		modals[this.type] = this;

		// Register websocket handler
		dispatcher[this.kind] = msg => this.render(msg[0]);
		// Send request
		main.send([this.kind]);
	},
	render(info) {
		if (!info.length) {
			return this.el.innerHTML = "God's in his Heaven. All's right"
				+ " with the world.";
		}
		this.el.innerHTML = this.renderContents(info);
		this.postRender && this.postRender()

		// Scroll to the end of the log
		this.el.scrollTop = this.el.scrollHeight;
		return this;
	},
	kill() {
		delete dispatcher[this.kind];
		delete modals[this.type];
		this.remove();
	}
});

const ModLogView = PanelView.extend({
	type: 'log',
	kind: common.MOD_LOG,
	renderContents(info) {
		return table(info, act => [
			// Unbans do not have a target post
			act.num ? oneeSama.postRef(act.num, act.op).safe : '',
			lang.mod.formatLog(act),
			oneeSama.time(act.time)
		]);
	}
});
exports.log = ModLogView;

const AdminPanelView = PanelView.extend({
	type: 'adminPanel',
	id: 'adminPanel',
	kind: common.ADMIN_PANEL,
	events: {
		'click .unban': 'unban'
	},
	renderContents(info) {
		this.banCount = 0;
		return table(info, ban => {
			this.banCount++;
			return [
				oneeSama.mnemonic(ban[0]),
				oneeSama.time(ban[1]),
				`<a class="unban" data-id="${ban[0]}">${lang.mod.unban}</a>`
			];
		});
	},
	unban(event) {
		const el = event.target;
		main.send([common.UNBAN, el.getAttribute('data-id')]);
		el.parentElement.parentElement.remove();

		// Check if any bans left
		if (!--this.banCount)
			this.kill();
	}
});
exports.adminPanel = AdminPanelView;

/**
 * Construct a table from an array of items and a consumer funtion that returns
 * an array of column contents.
 */
function table(rows, func) {
	let html = '<table>';
	for (let row of rows) {
		html += '<tr>';
		for (let cell of func(row)) {
			html += `<td>${cell}</td>`;
		}
		html += '</tr>';
	}
	html += '</table>';
	return html;
}

const RequestPanelView = PanelView.extend({
	type: 'djPanel',
	events: {
		'click .close': 'removeRequest',
		'click #rescan': 'scan'
	},

	/**
	 * Override parent method, because we don't query the server
	 */
	initialize() {
		main.$overlay[0].append(this.el)
		modals[this.type] = this;
		this.removed = new Memory('request', 2)
	    this.scan()
	},

	/**
	 * Scan thread for `/r/ song` strings, we have not removed yet
	 */
	scan() {
	    const removed = this.removed.readAll(),
			requests = []
		for (let {attributes} of state.posts.models) {
			// Post's request(s) already processed
			if (attributes.num in removed)
				continue
		    const m = attributes.body.match(/\/r\/[^\n]+$/gm)
			if (!m)
			    continue
			for (let request of m) {
			    request = request.replace('/r/', '').trim()
				requests.push([attributes.num, attributes.mnemonic, request])
			}
		}
		return this.render(_.sortBy(requests, 1))
	},

	/**
	 * Render the inner table
	 */
	renderContents(requests) {
		return table(requests, request => [
			oneeSama.postRef(request[0], oneeSama.op).safe,
			oneeSama.mnemonic(request[1]),
			request[2],
			`<a class="close" data-id="${request[0]}">X</a>`
		])
	},

	/**
	 * Extra rendering operations to perform that deviate from parent class
	 */
	postRender() {
	    this.el.append(etc.parseDOM(`<a id="rescan">${lang.rescan}</a>`)[0])
	},

	/**
	 * Remove a request from the list and persist removal to localStorage
	 */
	removeRequest(event) {
		event.preventDefault()
		const el = event.target
		this.removed.write(el.getAttribute('data-id'))
		el.closest('tr').remove()
	}
})
exports.djPanel = RequestPanelView
