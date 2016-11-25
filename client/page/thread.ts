import { escape } from '../util'
import { ThreadData, PostData, Post, OP } from '../posts/models'
import PostView, { OPView } from '../posts/view'
import { page, posts as postCollection, hidden } from '../state'
import { threads, importTemplate } from '../render'
import options from "../options"
import { setTitle } from "../tab"
import { expandAll } from "../posts/images"
import lang from "../lang"
import { updateSyncTimestamp } from "../connection"

// Container for all rendered posts
export let threadContainer: HTMLElement

// Render the HTML of a thread page
export default function renderThread(thread: ThreadData) {

	// TODO: Extract board configuration from HTML

	updateSyncTimestamp()
	const frag = importTemplate("thread")

	// Apply title to header and tab
	const title = `/${page.board}/ - ${escape(thread.subject)} (#${thread.id})`
	setTitle(title)
	frag.querySelector("h1").innerHTML = title

	threadContainer = frag.querySelector("#thread-container")
	if (!options.workModeToggle && (options.userBG || options.illyaDance)) {
		threadContainer.classList.add("custom-BG")
	}
	const els: Element[] = [],
		{posts} = thread
	delete thread.posts // Reduce strain on the GC. We won't be using these.

	frag.querySelector("#expand-images")
		.textContent = lang.ui[expandAll ? "contract" : "expand"]

	const opModel = new OP(thread),
		opView = new OPView(opModel)
	els.push(opView.el)
	postCollection.addOP(opModel)

	for (let post of posts) {
		if (!hidden.has(post.id)) {
			els.push(createPost(post))
		}
	}
	threadContainer.append(...els)

	if (page.lastN) {
		opView.renderOmit()
	}

	threads.innerHTML = ""
	threads.append(frag)
}

function createPost(data: PostData): Element {
	const model = new Post(data),
		view = new PostView(model)
	postCollection.add(model)
	return view.el
}
