import View from '../view'
import {Post, OP} from './models'
import {mine, posts, page} from '../state'
import {makeFrag, pluralize, HTML} from '../util'
import renderPost from './render/posts'
import {parseOpenLine, parseTerminatedLine} from './render/body'
import {write, read, importTemplate} from '../render'
import {renderBacklinks} from './render/etc'
import {posts as lang, navigation} from '../lang'

// Base post view class
export default class PostView extends View<Post> {
	// Only exist on open posts
	$buffer: Node        // Text node being written to
	$blockquote: Element // Entire text body of post
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
		const frag = importTemplate("article")
		renderPost(frag, this.model)
		if (this.model.editing) {
			this.$blockquote = frag.querySelector("blockquote")
			this.$lastLine = this.$blockquote.lastElementChild
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
		this.unbind()
		super.remove()
	}

	// Remove the model's cross references, but don't remove the element from
	// the DOM
	unbind() {
		this.model.view = this.model = null
	}

	// Replace the current line with a reparsed fragment
	reparseLine() {
		const frag = makeFrag(parseOpenLine(this.model.state)),
			line = frag.querySelector("span")
		this.findBuffer(frag)
		write(() => {
			this.$lastLine.replaceWith(frag)
			this.$lastLine = line
		})
	}

	// Start the new line as a quote
	startQuote() {
		const em = document.createElement("em")
		this.$buffer = document.createTextNode(">")
		em.append(this.$buffer)
		write(() =>
			this.$blockquote.append(em))
	}

	// Append a string to the current text buffer
	appendString(s: string) {
		write(() =>
			this.$buffer.textContent += s)
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
			this.$blockquote.append(this.$lastLine)
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

	// Close an open post and clean up
	closePost() {
		const html = parseTerminatedLine(this.model.state.line, this.model)
		write(() => {
			this.el.classList.remove("editing")
			this.$lastLine.innerHTML = html
			this.$lastLine = this.$buffer = this.$blockquote = null
		})
	}
}

// View of a threads opening post. Contains some extra functionality.
export class OPView extends PostView {
	$omit: Element
	model: OP

	constructor(model: Post) {
		super(model)
	}

	// Also attach the omitted post and image indicator
	render() {
		super.render()
		this.$omit = document.createElement("span")
		this.$omit.setAttribute("class", "omit")
		this.el.append(this.$omit)
	}

	// Render posts and images omited indicator
	renderOmit() {
		let images = 0,
			replies = -1
		for (let id in posts.models) {
			replies++
			if (posts.models[id].image) {
				images++
			}
		}

		const {imageCtr, postCtr} = this.model,
			imageOmit = imageCtr - images,
			replyOmit = postCtr - replies
		if (replyOmit === 0) {
			return
		}
		let html = pluralize(replyOmit, lang.post)
		if (imageOmit !== 0) {
			html += ` ${lang.and} ${pluralize(imageOmit, lang.image)} `
		}
		html += HTML
			`<span class="act">
				<a href="${page.href.split("?")[0]}" class="history">
					${navigation.seeAll}
				</a>
			<span>`
		write(() =>
			this.$omit.innerHTML = html)
	}
}
