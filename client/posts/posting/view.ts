import PostView from "../view"
import FormModel from "./model"
import { Post } from "../models"
import { boardConfig } from "../../state"
import { setAttrs, makeFrag } from "../../util"
import { parseTerminatedLine } from "../render/body"
import { renderHeader, renderName } from "../render/posts"
import { write, importTemplate } from "../../render"
import { threadContainer } from "../../page/thread"
import { postSM, postEvent } from "./main"
import UploadForm from "./upload"
import identity from "./identity"
import { atBottom, scrollToBottom } from "../../scroll"

// Element at the bottom of the thread to keep the fixed reply form from
// overlapping any other posts, when scrolled till bottom
let bottomSpacer: HTMLElement

// Post creation and update view
export default class FormView extends PostView {
    public model: FormModel
    private inputLock: boolean
    private input: HTMLElement
    private done: HTMLElement
    public cancel: HTMLElement
    private observer: MutationObserver
    private postControls: Element
    private previousHeight: number
    public upload: UploadForm

    constructor(model: Post, isOP: boolean) {
        super(model, null)
        this.renderInputs(isOP)
        if (!isOP) {
            this.el.classList.add("reply-form")
            this.initDraft()
        }
    }

    // Render extra input fields for inputting text and optionally uploading
    // images
    private renderInputs(isOP: boolean) {
        this.input = document.createElement("span")
        setAttrs(this.input, {
            id: "text-input",
            name: "body",
            contenteditable: "",
        })

        // Always make sure the input span always has at least 1 character, so
        // it does not float onto the image, if any.
        this.input.textContent = "\u200b"
        this.input.addEventListener("input", (event: Event) => {
            event.stopImmediatePropagation()
            this.onInput((event.target as Element).textContent)
        })
        this.input.addEventListener("keydown", (event: KeyboardEvent) =>
            this.onKeyDown(event))

        this.postControls = importTemplate("post-controls").firstElementChild
        this.el.querySelector(".post-container").append(this.postControls)

        this.done = this.el.querySelector("input[name=done]")
        this.done.addEventListener("click", postSM.feeder(postEvent.done))
        this.cancel = this.el.querySelector("input[name=cancel]")
        this.cancel.addEventListener("click", postSM.feeder(postEvent.done))

        if (isOP) {
            this.showDone()
        } else {
            if (!boardConfig.textOnly) {
                this.upload = new UploadForm(this.model, this.el)
                this.upload.input.addEventListener("change", () =>
                    this.model.uploadFile())
            }
            this.renderIdentity()
        }

        write(() => {
            this.blockquote.innerHTML = ""
            this.blockquote.append(this.input)
            this.input.focus()
        })
    }

    // Render a temporary view of the identity fields, so the user can see what
    // credentials he is about to post with
    public renderIdentity() {
        let {name} = identity,
            trip = ""
        const iHash = name.indexOf("#")
        if (iHash !== -1) {
            trip = "?"
            name = name.slice(0, iHash)
        }
        renderName(this.el.querySelector(".name"), {
            trip,
            name: name.trim(),
            auth: undefined,
        })
    }

    // Show button for closing allocated posts
    private showDone() {
        this.cancel.hidden = true
        this.done.hidden = false
    }

    // Initialize extra elements for a draft unallocated post
    private initDraft() {
        this.el.querySelector("header").classList.add("temporary")
        bottomSpacer = document.getElementById("bottom-spacer")

        // Keep this post and bottomSpacer the same height
        this.observer = new MutationObserver(() =>
            write(() =>
                this.resizeSpacer()))
        this.observer.observe(this.el, {
            childList: true,
            attributes: true,
            characterData: true,
            subtree: true,
        })

        write(() => {
            threadContainer.append(this.el)
            this.input.focus()
            this.resizeSpacer()
        })
    }

    // Resize bottomSpacer to the same top position as this post
    private resizeSpacer() {
        // Not a reply
        if (!bottomSpacer) {
            return
        }

        const {height} = this.el.getBoundingClientRect()
        // Avoid needless writes
        if (this.previousHeight === height) {
            return
        }
        this.previousHeight = height
        bottomSpacer.style.height = `calc(${height}px - 2.1em)`
    }

    private removeUploadForm() {
        write(() => {
            this.upload.input.remove()
            this.upload.status.remove()
        })
    }

    // Handle input events on input
    private onInput(val: string = this.input.textContent) {
        if (this.inputLock) {
            return
        }
        if (val === "") {
            this.lockInput(() =>
                this.input.textContent = "\u200b")
        }
        this.model.parseInput(val.replace("\u200b", ""))
    }

    // Ignore any oninput events on input during supplied function call
    private lockInput(fn: () => void) {
        this.inputLock = true
        fn()
        this.inputLock = false
    }

    // Handle keydown events on input
    private onKeyDown(event: KeyboardEvent) {
        if (event.which === 13) { // Enter
            event.preventDefault()
            return this.onInput(this.model.inputState.line + "\n")
        }
    }

    // Trim input from the end by the supplied length
    public trimInput(length: number) {
        let val = this.input.textContent.slice(0, -length) || "\u200b"
        write(() =>
            this.lockInput(() =>
                this.input.textContent = val))
    }


    // Replace the current line and set the cursor to the input's end
    public replaceLine(line: string) {
        write(() => {
            this.input.textContent = line || "\u200b"
            const range = document.createRange(),
                sel = window.getSelection()
            range.setEndAfter(this.input.lastChild)
            range.collapse(false)
            sel.removeAllRanges()
            sel.addRange(range)
            this.onInput()
        })
    }

    // Start a new line in the input field and close the previous one
    public startNewLine() {
        const {line} = this.model.inputState,
            frag = makeFrag(parseTerminatedLine(line, this.model))
        write(() => {
            this.input.before(frag)
            this.lockInput(() =>
                this.input.textContent = "\u200b")
        })
    }

    // Inject lines before input and set input contents to the lastLine
    public injectLines(lines: string[], lastLine: string) {
        const frag = document.createDocumentFragment()
        for (let line of lines) {
            const el = makeFrag(parseTerminatedLine(line, this.model))
            frag.append(el)
        }
        write(() =>
            this.input.before(frag))
        this.replaceLine(lastLine)
    }

    // Parse and replace the temporary like closed by input with a proper
    // parsed line
    public terminateLine(num: number) {
        const html = parseTerminatedLine(this.model.lastBodyLine(), this.model),
            frag = makeFrag(html)
        write(() =>
            this.blockquote.children[num].replaceWith(frag))
    }

    // Transform form into a generic post. Removes any dangling form controls
    // and frees up references.
    public cleanUp() {
        this.el.classList.remove("reply-form")
        if (this.postControls) {
            this.postControls.remove()
        }
        if (bottomSpacer) {
            bottomSpacer.style.height = ""
            if (atBottom) {
                scrollToBottom()
            }
        }
        if (this.observer) {
            this.observer.disconnect()
        }
        this.postControls
            = bottomSpacer
            = this.observer
            = this.done
            = this.cancel
            = this.input
            = this.upload
            = null
    }

    // Clean up on form removal
    public remove() {
        super.remove()
        this.cleanUp()
    }

    // Lock the post form after a critical error occurs
    public renderError() {
        write(() =>
            (this.el.classList.add("errored"),
                this.input.setAttribute("contenteditable", "false")))
    }

    // Transition into allocated post
    public renderAlloc() {
        this.id = "p" + this.model.id
        const header = this.el.querySelector("header")
        write(() => {
            this.el.id = this.id as string
            header.classList.remove("temporary")
            renderHeader(header, this.model)
            this.showDone()
        })
    }

    // Toggle the spoiler input checkbox
    public toggleSpoiler() {
        // Can only turn a spoiler on, if image already allocated
        if (this.model.image && this.model.image.spoiler) {
            return
        }

        write(() => {
            const el = this.el
                .querySelector("input[name=spoiler]") as HTMLInputElement
            el.checked = !el.checked
        })
    }

    // Insert image into the open post
    public insertImage() {
        this.renderImage(false, true)
        this.removeUploadForm()

        const {spoiler} = this.upload
        if (this.model.image.spoiler) {
            write(() =>
                spoiler.remove())
        } else {
            const fn = () =>
                this.upload.spoilerImage()
            spoiler.addEventListener("change", fn, {
                passive: true,
            })
        }
    }
}
