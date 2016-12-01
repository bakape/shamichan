import { Post } from './models'
import { makeFrag } from '../util'
import renderPost, { renderName, renderTime } from './render/posts'
import { parseOpenLine, parseTerminatedLine } from './render/body'
import { write, importTemplate } from '../render'
import { renderBacklinks } from './render/etc'
import ImageHandler from "./images"
import { ViewAttrs } from "../view"

// Base post view class
export default class PostView extends ImageHandler {
    // Only exist on open posts
    private buffer: Node        // Text node being written to
    protected blockquote: Element // Entire text body of post

    constructor(model: Post, el: HTMLElement) {
        const attrs: ViewAttrs = { model }
        if (el) {
            attrs.el = el
        } else {
            attrs.class = 'glass'
            if (model.editing) {
                attrs.class += ' editing'
            }
            attrs.tag = "article"
            attrs.id = "p" + model.id
        }
        super(attrs)

        this.model.view = this
        if (!el) {
            this.render()
        	this.autoExpandImage()
        }
    }

    // Render the element contents, but don't insert it into the DOM
    protected render() {
        const frag = importTemplate("article")
        this.renderContents(frag)
        this.el.append(frag)
    }

    // Render post into a container and find buffer positions
    public renderContents(container: DocumentFragment) {
        renderPost(container, this.model)
        if (this.model.editing) {
            this.blockquote = container.querySelector("blockquote")
            let buf = this.blockquote.lastChild
            if (!buf) {
                this.buffer = document.createElement("span")
                this.blockquote.append(this.buffer)
            } else {
                this.findBuffer(buf)
            }
        }
    }

    // Find the text buffer in an open line
    private findBuffer(b: Node) {
        const {state} = this.model
        if (state.quote) {
            b = b.lastChild
        }
        if (state.spoiler) {
            b = b.lastChild
        }
        if (!b) {
            b = this.lastLine()
        }
        this.buffer = b
    }

    // Remove the element from the DOM and detach from its model, allowing the
    // PostView instance to be garbage collected
    public remove() {
        this.unbind()
        super.remove()
    }

    // Remove the model's cross references, but don't remove the element from
    // the DOM
    public unbind() {
        this.model.view = this.model = null
    }

    // Replace the current line with a reparsed fragment
    public reparseLine() {
        const frag = makeFrag(parseOpenLine(this.model.state))
        this.findBuffer(frag.firstChild)
        write(() =>
            this.replaceLastLine(frag))
    }

    // Return the last line of the text body
    private lastLine(): Element {
        const ch = this.blockquote.children
        return ch[ch.length - 1]
    }

    // Replace the contents of the last line, accounting for the possibility of
    // there being no lines
    private replaceLastLine(node: Node) {
        const ll = this.lastLine()
        if (ll) {
            ll.replaceWith(node)
        } else {
            this.blockquote.append(node)
        }
    }

    // Append a string to the current text buffer
    public appendString(s: string) {
        write(() =>
            this.buffer.append(s))
    }

    // Remove one character from the current buffer
    public backspace() {
        write(() => {
            // Merge multiple successive nodes created by appendString()
            this.buffer.normalize()
            this.buffer.textContent = this.buffer.textContent.slice(0, -1)
        })
    }

    // Start a new line and reparse the old one
    public startNewLine() {
        const line = this.model.state.line.slice(0, -1),
            frag = makeFrag(parseTerminatedLine(line, this.model))
        write(() => {
            this.replaceLastLine(frag)
            this.buffer = document.createElement("span")
            this.blockquote.append(this.buffer)
        })
    }

    // Render links to posts linking to this post
    public renderBacklinks() {
        const html = renderBacklinks(this.model.backlinks)
        write(() =>
            this.el.querySelector("small").innerHTML = html)
    }

    // Close an open post and clean up
    public closePost() {
        const html = parseTerminatedLine(this.model.state.line, this.model),
            frag = makeFrag(html)
        write(() => {
            this.el.classList.remove("editing")
            this.replaceLastLine(frag)
            this.buffer = this.blockquote = null
        })
    }

    // Render the name and tripcode in the header
    public renderName() {
        write(() =>
            renderName(this.el.querySelector(".name"), this.model))
    }

    // Render the <time> element in the header
    public renderTime() {
        renderTime(this.el.querySelector("time"), this.model.time, false)
    }

    // Add highlight to post because it linked a post the client made, the
    // client made it or similar.
    public addHighlight() {
        write(() =>
            this.el.classList.add("highlight"))
    }
}
