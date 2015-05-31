/*
 * Extact model data from the thread tree HTML and populate models and views
 */

var $ = require('jquery'),
	main = require('./main'),
	state = require('./state'),
	posts = require('./posts');

class Extract {
	constructor() {
		// Read serialised model data
		const json = JSON.parse(main.$threads.children('#postData').text());
		main.command('notify:title', json.title);

		// We don't need models on catalog pages
		if (state.page.get('catalog'))
			return;

		this.mine = state.mine.read_all();
		this.posts = json.posts;
		let self = this;
		main.$threads.children('section').each(function() {
			self.extractThread($(this));
		});
	}
	extractThread($section) {
		let self = this;
		$section.children('article').each(function() {
			new posts.Article({
				model: new posts.models.Post(self.extractModel(this)),
				el: this
			});
		});
		// Extract the model of the OP
		let model = this.extractModel($section[0]);
		new posts.Section({
			model: new posts.models.Thread(model),
			el: $section[0]
		});
		/*
		 * Read the sync ID of the thread. Used later for syncronising with the
		 * server.
		 */
		state.syncs[model.num] = parseInt(model.hctr || 0, 10);
	}
	extractModel(el) {
		let info = this.posts[el.getAttribute('id')];
		// Did I make this post?
		if (this.mine[info.num])
			info.mine = true;
		return info;
	}
}
module.exports = Extract;
