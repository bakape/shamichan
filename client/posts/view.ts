import View, {ViewAttrs} from '../view'
import {Post} from './models'
import {mine} from '../state'
import {makeFrag} from '../util'
import renderPost from './render/posts'
import {write} from '../render'

// Base post view class
export default class PostView extends View<Post> {
	$buffer: Node // Contains the current line being edited, if any
	$blockQoute: Element

	constructor(model: Post) {
		let cls = 'glass'
		if (model.editing) {
			cls += ' editing'
		}
		if (mine.has(model.id)) {
			cls += ' highlight'
		}
		super({
			model,
			id: "p" + model.id,
			tag: "article",
			class: cls,
		})
		this.model.view = this
		this.render()
	}

	// Render the element contents, but don't insert it into the DOM
	render() {
		const frag = makeFrag(renderPost(this.model))
		let $b = this.$blockQoute = frag.querySelector("blockqoute")
		if (this.model.state.quote) {
			$b = $b.querySelector("em:last-of-type")
		}
		if (this.model.state.spoiler) {
			$b = $b.querySelector("del:las-of-type")
		}
		this.$buffer = $b
		this.el.append(frag)
	}

	// Remove the element from the DOM and detach from its model, allowing the
	// PostView instance to be garbage collected
	remove() {
		delete this.model.view
		delete this.model
		super.remove()
	}

	// Start the new line as a quote
	startQuote() {
		const em = document.createElement("em")
		this.$buffer = document.createTextNode(">")
		em.append(this.$buffer)
		write(() =>
			this.$blockQoute.append(em))
	}

	// Insert either an opening or closing spoiler tag in the $buffer
	insertSpoilerTag() {
		const {state} = this.model,
			$b = document.createTextNode("")
		if (!state.spoiler) {
			const del = document.createElement("del")
			del.append($b)
			const moveBuffer =
				state.quote
				? () =>
					this.$buffer.after(del)
				: () =>
					this.$blockQoute.append(del)
			write(() => {
				this.$buffer.textContent = state.line.slice(-1)
				moveBuffer()
				this.$buffer = $b
			})
		} else {
			if (!state.quote) {
				this.$buffer = this.$buffer.parentNode
			} else {
				this.$buffer = this.$blockQoute
			}
		}
	}

	// Append a string to the current text buffer
	appendString(s: string) {
		write(() =>
			this.$buffer.append(s))
	}
}
