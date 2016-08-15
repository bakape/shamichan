import {HTML, escape, makeFrag} from '../util'
import {navigation as lang} from '../lang'
import {ThreadData, PostData, Post} from '../posts/models'
import PostView, {OPView} from '../posts/view'
import {page, posts} from '../state'
import {write, $threads, importTemplate} from '../render'
import renderPost from '../posts/render/posts'

// Container for all rendered posts
export let $threadContainer: Element

// Render the HTML of a thread page
export default function renderThread(thread: ThreadData) {

	// TODO: Apply thread title as tab title

	const frag = importTemplate("thread"),
		title = `/${page.board}/ - ${escape(thread.subject)} (#${thread.id})`

	frag.querySelector("h1").textContent = title

	$threadContainer = frag.querySelector("#thread-container")
	const els: Element[] = [],
		{posts} = thread
	delete thread.posts // Reduce strain on the GC. We won't be usng these.

	const opModel = new Post(thread),
		opView = new OPView(opModel)
	els.push(opView.el)

	for (let id in posts) {
		els.push(createPost(thread.posts[id]))
	}
	$threadContainer.append(...els)

	if (page.lastN) {
		opView.renderOmit()
	}

	write(() => {
		$threads.innerHTML = ""
		$threads.append(frag)
	})
}

function createPost(data: PostData): Element {
	const model = new Post(data),
		view = new PostView(model)
	posts.add(model)
	return view.el
}
