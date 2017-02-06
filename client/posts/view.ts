import { Post } from './model'
import { makeFrag, importTemplate } from '../util'
import { renderPost, renderName, renderTime, renderBanned, parseBody, renderBacklinks } from './render'
import ImageHandler from "./images"
import { ViewAttrs } from "../base"

// Base post view class
export default class PostView extends ImageHandler {
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
        renderPost(this.el, this.model)
    }

    // Get the current Element for text to be written to
    private buffer(): Element {
        const {state: {spoiler, quote}} = this.model
        let buf = this.el.querySelector("blockquote") as Element
        if (quote) {
            buf = buf.lastElementChild
        }
        if (spoiler) {
            buf = buf.lastElementChild
        }
        return buf
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

    // Replace the current body with a reparsed fragment
    public reparseBody() {
        if (!this.model.body) {
            return
        }
        const frag = makeFrag(parseBody(this.model))
        this.replaceBody(frag)
    }

    // Replace the text body of the post
    private replaceBody(node: Node) {
        const bq = this.el.querySelector("blockquote")
        bq.innerHTML = ""
        bq.append(node)
    }

    // Append a string to the current text buffer
    public appendString(s: string) {
        this.buffer().append(s)
    }

    // Remove one character from the current buffer
    public backspace() {
        const buf = this.buffer()
        // Merge multiple successive nodes created by appendString()
        buf.normalize()
        buf.innerHTML = buf.innerHTML.slice(0, -1)
    }

    // Render links to posts linking to this post
    public renderBacklinks() {
        renderBacklinks(this.el, this.model.backlinks)
    }

    // Close an open post and clean up
    public closePost() {
        const frag = makeFrag(parseBody(this.model))
        this.el.classList.remove("editing")
        this.replaceBody(frag)
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
        renderBanned(this.el)
    }

    // Add or remove highlight to post
    public setHighlight(on: boolean) {
        this.el.classList.toggle("highlight", on)
    }
}
