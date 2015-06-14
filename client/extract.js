/*
 * Extact model data from the thread tree HTML and populate models and views
 */

let main = require('./main'),
	{_, options, state, posts} = main;

class Extract {
	constructor() {
		let el = main.$threads[0];
		// Read serialised model data
		let json = JSON.parse(document.getElementById('postData').innerHTML);
		main.command('notify:title', json.title);

		// We don't need models on catalog pages
		if (state.page.get('catalog'))
			return;

		this.mine = state.mine.read_all();
		this.posts = json.posts;
		this.extractReplies(el);
		this.extractThreads(el);

		// Apply various client-only DOM modifications
		_.defer(function() {
			if (options.get('anonymise'))
				main.command('loop:anonymise');
			main.command('time:render');
		})
	}
	extractReplies(el) {
		let articles = el.getElementsByTagName('article'),
			Article = posts.Article,
			Post = posts.models.Post;
		for (let i = 0, l = articles.length; i < l; i++) {
			let article = articles[i];
			new Article({
				model: new Post(this.extractModel(article)),
				el: article
			});
		}
	}
	extractThreads(el) {
		let sections = el.getElementsByTagName('section'),
			Section = posts.Section,
			Thread = posts.models.Thread,
			syncs = state.syncs;
		for (let i = 0, l = sections.length; i < l; i++) {
			let section = sections[i];
			const model = this.extractModel(section);
			new Section({
				model: new Thread(model),
				el: section
			});
			// Read the sync ID of the thread. Used later for syncronising
			// with the server.
			syncs[model.num] = model.hctr;
		}
	}
	extractModel(el) {
		let info = this.posts[el.getAttribute('id')];
		// Did I make this post?
		if (info.num in this.mine)
			info.mine = true;
		return info;
	}
}
module.exports = Extract;

// Initial extraction. No need to defer, as we actually want it to hit ASAP.
new Extract();
