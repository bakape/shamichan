import { posts, page } from "../state"
import { on, fetchJSON } from "../util"
import options from "../options"
import { Post } from "./model"
import { PostData } from "../common"
import PostView from "./view"
import PostCollection from "./collection"

// Stored models of posts, that have been created with inline expansion. This
// lets these models still be queried by certain functions, that expect a
// model-view pair.
export const inlinedPosts = new PostCollection()

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

	if (parent.lastElementChild.tagName === "ARTICLE") {
		return contractPost(id, parent)
	}

	let model = posts.get(id) || inlinedPosts.get(id),
		found = false
	if (model) {
		// Can not create cyclic DOM trees
		if (model.view.el.contains(parent)) {
			return
		}
		found = true

		// Remove references, if already inlined
		const oldParent = model.view.el.parentElement
		if (oldParent.tagName === "EM") {
			toggleLinkReferences(oldParent, id, false)
		}
	} else {
		// Fetch external post from server
		const [data] = await fetchJSON<PostData>(`/json/post/${id}`)
		if (data) {
			model = new Post(data)
			new PostView(model, null)
			found = true
			inlinedPosts.add(model)
		}
	}

	if (found) {
		parent.append(model.view.el)
		toggleLinkReferences(parent, id, true)
	}
}

// contract and already expanded post and return it to its former position
function contractPost(id: number, parent: HTMLElement) {
	toggleLinkReferences(parent, id, false)

	const model = posts.get(id)
	if (!model) {
		// Fetched from the server and not originally part of the thread
		inlinedPosts.get(id).remove()
	} else {
		model.view.reposition()
	}
}

// Highlight or unhighlight links referencing the parent post in the child post
function toggleLinkReferences(parent: Element, childID: number, on: boolean) {
	const p = parent.closest("article"),
		ch = document.getElementById(`p${childID}`),
		pID = p.closest("article").id.slice(1)
	for (let el of p.querySelectorAll(".post-link")) {
		// Check, if not from a post inlined in the child
		if (el.closest("article") === ch
			&& el.getAttribute("data-id") === pID
		) {
			el.classList.toggle("referenced", on)
		}
	}
}

export default () => {
	// Clear, when changing the page
	page.onChange("thread", () =>
		inlinedPosts.clear())

	on(document.getElementById("threads"), "click", onClick, {
		selector: ".post-link",
	})
}

