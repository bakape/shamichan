import PostCollection from "./collection"
import { View } from "../base"
import PostView from "./view"
import { PostData } from "../common"
import { Post } from "./model"
import { posts } from "../state"

const overlay = document.getElementById("modal-overlay")

// Displays a collection of posts in a floating modal
export default class CollectionView extends View<PostCollection> {
	public model = new PostCollection()
	private borrowed: Post[] = [] // Already exists in the DOM

	constructor(data: PostData[]) {
		super({
			tag: "div",
			class: "modal post-collection",
		})

		// Close button
		const closer = document.createElement("a")
		closer.textContent = `[X]`
		closer.style.cssFloat = "right"
		this.el.append(closer)
		closer.addEventListener("click", this.remove.bind(this), {
			passive: true,
		})

		// Append posts to view
		data = data.sort((a, b) =>
			a.id - b.id)
		for (let d of data) {
			let model = PostCollection.getFromAll(d.id)
			if (!model) {
				model = new Post(d)
				new PostView(model, null)
				this.model.add(model)
			} else {
				this.borrowed.push(model)
			}
			this.el.append(model.view.el)
		}

		overlay.append(this.el)
		this.el.style.display = "block"
	}

	public remove() {
		for (let m of this.borrowed) {
			// TODO: Method for returning cross-thread inlined posts back into
			// their old position
			if (posts.get(m.id)) {
				m.view.reposition()
			}
		}
		this.model.unregister()
		super.remove()
	}
}
