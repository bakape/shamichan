/*
General post backbone models
 */

let main = require('../main'),
	{_, Backbone, state} = main;

exports.Post = Backbone.Model.extend({
	idAttribute: 'num',
	initialize() {
		this.initCommon();
	},
	// Initialisation logic common to both replies and threads
	initCommon() {
		state.posts.add(this);
		const links = this.get('links');
		if (links)
			this.forwardLinks(null, links);
		this.listenTo(this, 'change:links', this.forwardLinks);
	},
	// Proxy commands to the view(s). Using a central channel helps us reduce
	// listener count overhead.
	dispatch(command, ...args) {
		this.trigger('dispatch', command, ...args);
	},
	remove() {
		this.stopListening();
		// Remove view
		this.dispatch('remove');
		// Remove from post collection
		state.posts.remove(this);
	},
	// Calling a method is always less overhead than binding a dedicated
	// listener for each post's image
	setImage(image, silent) {
		this.set('image', image);
		if (!silent)
			this.dispatch('renderImage', image);
	},
	setSpoiler(spoiler) {
		let image = this.get('image');
		image.spoiler = spoiler;
		this.dispatch('renderImage', image);
	},
	removeImage() {
		this.unset('image');
		this.dispatch('renderImage', null);
	},
	addLinks(links){
		if(!links)
			return;
		var old = this.get('links');
		if(!old)
			return this.set({links: links});
		_.extend(old,links);
		this.set({links:old});
		// If we get here we changed something for sure, but as we are using the
		// same ref backbone will ignore it so we have to force the event to
		// trigger.
		this.trigger('change:links', this, old);
	},
	// Pass this post's links to the central model
	forwardLinks(model, links) {
		var old, newLinks;
		const num = this.get('num'),
			op = this.get('op') || num;
		const mine = state.mine.read_all();
		for (let key in links) {
			if (mine[key])
				main.command('repliedToMe', this);
			old = state.linkerCore.get(key);
			newLinks = old ? _.clone(old) : {};
			newLinks[num] = op;
			state.linkerCore.set(key, newLinks);
		}
	}
});

exports.Thread = exports.Post.extend({
	defaults: {
		replies: [],
		omit: 0,
		image_omit: 0
	},
	initialize() {
		// Omitted images can only be calculated, if there are omitted posts
		if (this.get('omit'))
			this.getImageOmit();
		this.initCommon();
	},
	remove() {
		this.stopListening();
		this.trigger('remove');
		state.posts.remove(this);

		// Propagate model removal to all replies
		const replies = this.get('replies');
		for (let i = 0, lim = replies.length; i < lim; i++) {
			let model = state.posts.get(replies[i]);
			if (model)
				model.remove();
		}
	},
	/*
	 With the current renderring and storage implementations we can not get the
	 image omit count during the server-side render.
	 */
	getImageOmit() {
		let image_omit = this.get('imgctr') -1;
		const replies = this.get('replies');

		for (let i = 0, lim = replies.length; i < lim; i++) {
			let model = state.posts.get(replies[i]);
			if (!model)
				continue;
			if (model.get('image'))
				image_omit--;
		}
		this.set('image_omit', image_omit);
	},
	toggleLocked(val) {
		this.dispatch('renderLocked', val);
	}
});
