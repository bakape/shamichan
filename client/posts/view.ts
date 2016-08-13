import View from '../view'
import {Post} from './models'
import {mine} from '../state'
import {makeFrag} from '../util'
import renderPost from './render/posts'
import {parseOpenLine, parseTerminatedLine} from './render/body'
import {write, read} from '../render'
import {renderBacklinks} from './render/etc'

// Base post view class
export default class PostView extends View<Post> {
	// Only exist on open posts
	$buffer: Node        // Text node being written to
	$blockQoute: Element // Entire text body of post
	$lastLine: Element   // Contains the current line being edited, if any

	constructor(model: Post) {
		let cls = 'glass'
		if (model.editing) {
			cls += ' editing'
		}

		let highlight: boolean
		if (mine.has(model.id)) {
			highlight = true
		} else if (model.links) {
			for (let id in model.links) {
				if (mine.has(parseInt(id))) {
					highlight = true
					break
				}
			}
		}
		if (highlight) {
			cls += ' highlight'
		}

		// TODO: If post has links to my posts, send desktop notifications. Best
		// integrate with a last post seen counter? Maybe we need to store a
		// "seen" status for all posts, but that would be a lot of overhead.

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
		if (this.model.editing) {
			this.$blockQoute = frag.querySelector("blockqoute")
			this.$lastLine = this.$blockQoute.lastElementChild
			this.findBuffer(this.$lastLine)
		}
		this.el.append(frag)
	}

	// Find the text buffer in an open line
	findBuffer($b: Node) {
		const {state} = this.model
		if (state.quote) {
			$b = $b.lastChild
		}
		if (state.spoiler) {
			$b = $b.lastChild
		}
		this.$buffer = $b
	}

	// Remove the element from the DOM and detach from its model, allowing the
	// PostView instance to be garbage collected
	remove() {
		delete this.model.view
		delete this.model
		super.remove()
	}

	// Replace the current line with a reparsed fragment
	reparseLine() {
		const frag = makeFrag(parseOpenLine(this.model.state))
		this.findBuffer(frag)
		write(() => {
			this.$lastLine.replaceWith(frag)
			this.$lastLine = frag as Element
		})
	}

	// Start the new line as a quote
	startQuote() {
		const em = document.createElement("em")
		this.$buffer = document.createTextNode(">")
		em.append(this.$buffer)
		write(() =>
			this.$blockQoute.append(em))
	}

	// Append a string to the current text buffer
	appendString(s: string) {
		write(() =>
			this.$buffer.append(s))
	}

	// Remove one character from the current buffer
	backspace() {
		write(() =>
			this.$buffer.textContent = this.$buffer.textContent.slice(0, -1))
	}

	// Start a new line and reparse the old one
	startNewLine() {
		const line = this.model.state.line.slice(0, -1),
			frag = makeFrag(parseTerminatedLine(line, this.model))
		write(() => {
			this.$lastLine.replaceWith(frag),
			this.$buffer = document.createTextNode("")
			this.$lastLine = document.createElement("span")
			this.$lastLine.append(this.$buffer)
			this.$blockQoute.append(this.$lastLine)
		})
	}

	// Render links to posts linking to this post
	renderBacklinks() {
		const html = renderBacklinks(this.model.backlinks)
		read(() => {
			const el = this.el.querySelector("small")
			write(() =>
				el.innerHTML = html)
		})
	}
}
