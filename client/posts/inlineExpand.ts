import { posts } from "../state"
import { renderFetchedPost } from "../fetch"
import { on } from "../util"
import { threads, write } from "../render"

// Expand or contract linked posts inline
async function onClick(e: MouseEvent) {
	// Don't trigger, when user is trying to open in a new tab
	if (e.which !== 1 || e.ctrlKey) {
		return
	}

	e.preventDefault()

	const el = e.target as Element,
		parent = el.parentElement,
		id = parseInt(el.getAttribute("data-id"))

	if (parent.classList.contains("expanded")) {
		return contractPost(id, parent)
	}

	const model = posts.get(id)
	if (model) {
		// Can not create recursive DOM trees
		if (model.view.el.contains(parent)) {
			return
		}

		return write(() => {
			parent.classList.add("expanded")
			parent.append(model.view.el)
		})
	}

	// Fetch external post from server
	const view = await renderFetchedPost(id)
	if (!view) {
		return
	}
	write(() => {
		parent.classList.add("expanded")
		parent.append(view.el)
	})
}

// contract and already expanded post and return it to its former position
function contractPost(id: number, parent: HTMLElement) {
	const model = posts.get(id)
	// Fetched from server and not originally part of the thread
	if (!model) {
		return write(() =>
			document.getElementById(`p${id}`).remove())
	}

	// Find the ID of the post preceding this one. Make sure the target post is
	// not expanded inline itself.
	const ids = Object.keys(posts.models).sort()
	let i = ids.indexOf(id.toString())
	while (true) {
		const previous = posts.get(parseInt(ids[i - 1]))
		if (!previous) {
			write(() =>
				document
					.getElementById("thread-container")
					.prepend(model.view.el))
			break
		}
		if (previous.view.el.matches("#thread-container > article")) {
			write(() =>
				previous.view.el.before(model.view.el))
			break
		}
		i--
	}
}

on(threads, "click", onClick, {
	selector: ".post-link",
})
