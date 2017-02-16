import { posts, page } from "../state"
import { on, fetchJSON } from "../util"
import options from "../options"
import { Post } from "./model"
import { PostData } from "../common"
import PostView from "./view"

// Expand or contract linked posts inline
async function onClick(e: MouseEvent) {
	const el = e.target as Element

	// Don't trigger, when user is trying to open in a new tab, inline
	// expansion is disabled or the link is temporary
	const bypass = e.which !== 1
		|| e.ctrlKey
		|| !options.postInlineExpand
		|| el.classList.contains("temp")
	if (bypass) {
		return
	}

	e.preventDefault()

	const parent = el.parentElement,
		id = parseInt(el.getAttribute("data-id"))

	if (parent.classList.contains("expanded")) {
		return contractPost(id, parent)
	}

	const model = posts.get(id)
	let found = false
	if (model && page.thread) {
		// Can not create cyclic DOM trees
		if (model.view.el.contains(parent)) {
			return
		}

		found = true
		parent.classList.add("expanded")
		parent.append(model.view.el)
	} else {
		// Fetch external post from server
		const [data] = await fetchJSON<PostData>(`/json/post/${id}`)
		if (data) {
			const model = new Post(data),
				view = new PostView(model, null)
			found = true
			parent.classList.add("expanded", "fetched")
			parent.append(view.el)
		}
	}

	if (found) {
		toggleLinkReferences(parent, id, true)
	}
}

// contract and already expanded post and return it to its former position
function contractPost(id: number, parent: HTMLElement) {
	const wasFetched = parent.classList.contains("fetched")
	parent.classList.remove("expanded", "fetched")
	const model = posts.get(id)
	// Fetched from server and not originally part of the thread or removed from
	// the thread
	if (wasFetched || !model) {
		return parent.lastElementChild.remove()
	}

	// Find the ID of the post preceding this one. Make sure the target post is
	// not expanded inline itself.
	const ids = Object.keys(posts.models).sort()
	let i = ids.indexOf(id.toString())
	while (true) {
		const previous = posts.get(parseInt(ids[i - 1]))
		if (!previous) {
			document.getElementById("thread-container").prepend(model.view.el)
			break
		}
		if (previous.view.el.matches("#thread-container > article")) {
			toggleLinkReferences(parent, id, false)
			previous.view.el.before(model.view.el)
			break
		}
		i--
	}
}

// Highlight or unhighlight links referencing the parent post in the child post
function toggleLinkReferences(parent: Element, childID: number, on: boolean) {
	const p = parent.closest("article"),
		ch = document.getElementById(`p${childID}`),
		pID = p.closest("article").id.slice(1)
	for (let el of p.querySelectorAll(".post-link")) {
		// Check if not from a post inlined in the child
		if (el.closest("article") === ch && el.getAttribute("data-id") == pID) {
			el.classList.toggle("referenced", on)
		}
	}
}

export default () =>
	on(document.getElementById("threads"), "click", onClick, {
		selector: ".post-link",
	})

