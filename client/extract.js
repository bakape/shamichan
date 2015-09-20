/*
 * Extact model data from the thread tree HTML and populate models and views
 */

const main = require('./main'),
	{_, options, state, posts} = main;

class Extract {
	constructor(catalog) {
		const el = main.$threads[0];

		// Read serialised model data
		const json = JSON.parse(document.getElementById('postData').innerHTML);
		main.request('notify:title', json.title);

		// We don't need models on catalog pages
		if (catalog)
			return;

		const mine = this.mine = state.mine.readAll(),
			posts = this.posts = json.posts;
		this.extractReplies(el);
		this.extractThreads(el);

		state.addLinks(json.links);
		// Forward posts that replied to my post
		for (let post in posts) {
			const links = posts[post].links;
			if (!links)
				continue;
			for (let num in links) {
				if (num in mine)
					main.request('repliedToMe', posts[post].num);
			}
		}

		// Apply various client-only DOM modifications
		if (options.get('anonymise'))
			main.request('loop:anonymise');
		main.request('time:render');
	}
	extractReplies(el) {
		let articles = el.getElementsByTagName('article');
		for (let i = 0, l = articles.length; i < l; i++) {
			let article = articles[i];
			new posts.Article({
				model: new posts.models.Post(this.extractModel(article)),
				el: article
			});
		}
	}
	extractThreads(el) {
		let sections = el.getElementsByTagName('section');
		for (let i = 0; i < sections.length ; i++) {
			let section = sections[i];
			const model = this.extractModel(section);
			new posts.Section({
				model: new posts.models.Thread(model),
				el: section
			})
				 .renderOmit();
			// Read the sync ID of the thread. Used later for syncronising
			// with the server.
			state.syncs[model.num] = model.hctr;
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
new Extract(state.page.get('catalog'));
