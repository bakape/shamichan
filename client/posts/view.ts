import { Post } from './model'
import { makeFrag, write, importTemplate } from '../util'
import {
    renderPost, renderName, renderTime, renderBanned, parseOpenLine,
    parseOpenBody, parseTerminatedLine, renderBacklinks
} from './render'
import ImageHandler from "./images"
import { ViewAttrs } from "../base"

// Base post view class
export default class PostView extends ImageHandler {
    // Text node being written to. Only exist on open posts
    private _buffer: Node

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
        this.el.append(importTemplate("article"))
        this.renderContents()
    }

    // Render post into a container and find buffer positions
    public renderContents() {
        renderPost(this.el, this.model)
    }

    // Render the text body of an open post
    public renderOpenBody() {
        const el = this.el.querySelector("blockquote")
        el.innerHTML = parseOpenBody(this.model)
    }

    // Get the current Node for text to be written to
    private buffer(): Node {
        if (this._buffer) {
            return this._buffer
        } else {
            this.findBuffer(this.lastLine())
            return this._buffer
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
        this._buffer = b
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
        return this.el.querySelector("blockquote").lastElementChild
    }

    // Replace the contents of the last line, accounting for the possibility of
    // there being no lines
    private replaceLastLine(node: Node) {
        const ll = this.lastLine()
        if (ll) {
            ll.replaceWith(node)
        } else {
            this.el.querySelector("blockquote").append(node)
        }
    }

    // Append a string to the current text buffer
    public appendString(s: string) {
        write(() =>
            this.buffer().append(s))
    }

    // Remove one character from the current buffer
    public backspace() {
        write(() => {
            const buf = this.buffer()
            // Merge multiple successive nodes created by appendString()
            buf.normalize()
            buf.textContent = buf.textContent.slice(0, -1)
        })
    }

    // Start a new line and reparse the old one
    public startNewLine() {
        const line = this.model.state.line.slice(0, -1),
            frag = makeFrag(parseTerminatedLine(line, this.model))
        write(() => {
            this.replaceLastLine(frag)
            this._buffer = document.createElement("span")
            this.el.querySelector("blockquote").append(this._buffer)
        })
    }

    // Render links to posts linking to this post
    public renderBacklinks() {
        write(() =>
            renderBacklinks(this.el, this.model.backlinks))
    }

    // Close an open post and clean up
    public closePost() {
        const html = parseTerminatedLine(this.model.state.line, this.model),
            frag = makeFrag(html)
        write(() => {
            this.el.classList.remove("editing")
            this.replaceLastLine(frag)
        })
    }

    // Render the name and tripcode in the header
    public renderName() {
        renderName(this.el.querySelector(".name"), this.model)
    }

    // Render the <time> element in the header
    public renderTime() {
        renderTime(this.el.querySelector("time"), this.model.time, false)
    }

    // Render ban notice on post
    public renderBanned() {
        write(() =>
            renderBanned(this.el))
    }

    // Add or remove highlight to post
    public setHighlight(on: boolean) {
        write(() =>
            this.el.classList.toggle("highlight", on))
    }
}
