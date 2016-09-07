import {escape} from '../util'
import {ThreadData, PostData, Post, OP} from '../posts/models'
import PostView, {OPView} from '../posts/view'
import {page, posts as postCollection} from '../state'
import {write, $threads, importTemplate} from '../render'
import options from "../options"

// Container for all rendered posts
export let $threadContainer: Element

// Render the HTML of a thread page
export default function renderThread(thread: ThreadData) {

	// TODO: Apply thread title as tab title

	const frag = importTemplate("thread"),
		title = `/${page.board}/ - ${escape(thread.subject)} (#${thread.id})`

	frag.querySelector("h1").innerHTML = title

	$threadContainer = frag.querySelector("#thread-container")
	if (options.userBG || options.illyaDance) {
		$threadContainer.classList.add("custom-BG")
	}
	const els: Element[] = [],
		{posts} = thread
	delete thread.posts // Reduce strain on the GC. We won't be usng these.

	// Render larger thumbnail for the OP
	if (thread.image) {
		thread.image.large = true
	}

	const opModel = new OP(thread),
		opView = new OPView(opModel)
	els.push(opView.el)
	postCollection.addOP(opModel)

	for (let id in posts) {
		els.push(createPost(posts[id]))
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
	postCollection.add(model)
	return view.el
}
