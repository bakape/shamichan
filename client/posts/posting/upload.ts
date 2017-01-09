import lang from '../../lang'
import { load, postJSON, postText, write } from '../../util'
import identity from "./identity"
import { Post } from "../model"
import { View } from "../../base"

// Precompute 00 - ff strings for conversion to hexadecimal strings
const precomputedHex = new Array(256)
for (let i = 0; i < 256; i++) {
    precomputedHex[i] = (i < 16 ? '0' : '') + i.toString(16)
}

// Uploaded file data to be embedded in thread and reply creation or file
// insertion requests
export type FileData = {
    token: string
    name: string
    spoiler?: boolean
}

interface LoadProgress {
    total: number
    loaded: number
}

// Mixin for handling file uploads
export default class UploadForm extends View<Post> {
    public spoiler: HTMLElement
    public status: HTMLElement
    public input: HTMLInputElement

    constructor(model: Post, parent: HTMLElement) {
        const el = parent.querySelector(".upload-container")
        el.hidden = false
        super({ el, model })
        this.spoiler = el.querySelector(`span[data-id="spoiler"]`)
        this.status = el.querySelector(".upload-status")
        this.input = el.querySelector("input[name=image]") as HTMLInputElement
    }

    // Read the file from input and send as a POST request to the server.
    // Returns image request data, if upload succeeded.
    public async uploadFile(
        file: File = this.input.files[0],
    ): Promise<FileData> {
        if (!navigator.onLine) {
            return null
        }

        let token: string

        write(() => {
            this.input.style.display = "none"
            this.renderProgress({
                total: 1,
                loaded: 0,
            })
        })

        // First send a an SHA1 hash to the server, in case it already has the
        // file thumbnailed and we don't need to upload.
        const r = new FileReader()
        r.readAsArrayBuffer(file)
        const {target: {result}} = await load(r) as ArrayBufferLoadEvent,
            hash = await crypto.subtle.digest("SHA-1", result),
            [res, err] = await postText("/uploadHash", bufferToHex(hash))
        if (err) {
            throw err
        }
        if (res) {
            token = res
        } else {
            // If there is none, upload file like normal
            token = await this.upload(file)
            if (!token) {
                return null
            }
        }

        const img: FileData = {
            token,
            name: file.name,
        }
        const spoiler =
            (this.el.querySelector("input[name=spoiler]") as HTMLInputElement)
                .checked
        if (spoiler) {
            img.spoiler = true
        }
        return img
    }

    // Upload file to server and return the file allocation token
    private async upload(file: File): Promise<string> {
        const formData = new FormData()
        formData.append("image", file)

        // Not using fetch, because no ProgressEvent support
        const xhr = new XMLHttpRequest()
        xhr.open("POST", "/upload")
        xhr.upload.onprogress = e =>
            this.renderProgress(e)
        xhr.send(formData)
        await load(xhr)

        if (xhr.status !== 200) {
            write(() => {
                this.status.textContent = xhr.response
                this.input.style.display = ""
            })
            return ""
        }

        return xhr.responseText
    }

    // Render client-side upload progress
    private renderProgress({total, loaded}: LoadProgress) {
        let s: string
        if (loaded === total) {
            s = lang.ui["thumbnailing"]
        } else {
            const n = Math.floor(loaded / total * 100)
            s = `${n}% ${lang.ui["uploadProgress"]}`
        }
        write(() =>
            this.status.textContent = s)
    }

    // Spoiler an image file after it has already been allocated
    public async spoilerImage() {
        await spoilerImage(this.model as Post)
        write(() =>
            this.spoiler.remove())
    }
}

// Spoiler a post's image.
export async function spoilerImage({id}: Post) {
    await postJSON("/spoiler", {
        id,
        password: identity.postPassword,
    })
}

// Encodes an ArrayBuffer to a hex string
function bufferToHex(buf: ArrayBuffer): string {
    const b = new Uint8Array(buf),
        res = new Array(buf.byteLength)
    for (let i = 0; i < res.length; i++) {
        res[i] = precomputedHex[b[i]]
    }
    return res.join('')
}
