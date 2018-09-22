import lang from '../../lang';
import { load, trigger } from '../../util';
import { Post } from "../model";
import { View } from "../../base";
import { config } from "../../state";
import { postSM, postEvent, postState } from ".";

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

// View for handling file uploads
export default class UploadForm extends View<Post> {
    private isUploading: boolean;
    private spoiler: HTMLElement;
    private button: HTMLElement;
    private hiddenInput: HTMLInputElement;
    private xhr: XMLHttpRequest;
    private bufferedFile: File; // In case we need to resubmit a file

    constructor(model: Post, el: HTMLElement) {
        super({ el, model });
        el.hidden = false;
        this.spoiler = el
            .querySelector(`span[data-id="spoiler"]`) as HTMLInputElement;
        this.hiddenInput = el
            .querySelector("input[name=image]") as HTMLInputElement;
        this.button = el.querySelector("button");

        this.button.addEventListener("click", () => {
            if (this.isUploading) {
                this.reset();
            } else if (this.canAllocImage()) {
                this.hiddenInput.click();
            }
        }, { passive: true });
        this.hiddenInput.addEventListener("change", () => {
            if (this.canAllocImage() && this.hiddenInput.files.length) {
                trigger("getPostModel").uploadFile(this.hiddenInput.files[0]);
            }
        }, { passive: true });
    }

    private canAllocImage(): boolean {
        switch (postSM.state) {
            case postState.draft:
            case postState.allocating:
            case postState.alloc:
                return true;
            default:
                return false;
        }
    }

    // Read the file from input and send as a POST request to the server.
    // Returns image request data, if upload succeeded.
    public async uploadFile(file: File): Promise<FileData> | null {
        if (!navigator.onLine || this.isUploading) {
            return null;
        }
        if (file.size > (config.maxSize << 20)) {
            this.reset(lang.ui["fileTooLarge"]);
            return null;
        }

        this.bufferedFile = file;
        this.isUploading = true;
        this.renderProgress({ total: 1, loaded: 0 });

        let token: string;
        // Detect, if the crypto API can be used
        if (location.protocol === "https:"
            || location.hostname === "localhost"
        ) {
            // First send a an SHA1 hash to the server, in case it already has
            // the file thumbnailed and we don't need to upload.
            const r = new FileReader();
            r.readAsArrayBuffer(file);
            const { target: { result } }
                = await load(r) as ArrayBufferLoadEvent;
            const res = await fetch("/api/upload-hash", {
                method: "POST",
                body: bufferToHex(await crypto.subtle.digest("SHA-1", result)),
            });
            const text = await res.text();
            if (this.handleResponse(res.status, text)) {
                token = text;
            } else {
                return null;
            }
        }

        if (!token) {
            token = await this.upload(file);
            if (!token) {
                this.isUploading = false;
                return null;
            }
        }

        this.isUploading = false;
        return {
            token,
            name: file.name,
            spoiler: this.inputElement("spoiler").checked,
        };
    }

    // Handle a server response and return, if the request succeeded.
    private handleResponse(code: number, text: string): boolean {
        switch (code) {
            case 200:
                return true;
            case 403:
                if (this.isCaptchaRequest(text)) {
                    postSM.feed(postEvent.captchaRequested);
                    this.reset();
                    return false;
                }
            // Retry on imager connectivity problems
            case 502:
                if (this.canAllocImage()) {
                    trigger("getPostModel").retryUpload();
                    this.reset();
                    return false;
                }
            default:
                this.reset(text);
                return false;
        }
    }

    // Display upload status with optional
    private displayStatus(status: string, title?: string) {
        this.button.textContent = status;
        this.button.title = title || "";
    }

    private isCaptchaRequest(s: string) {
        return s.indexOf("captcha required") !== -1;
    }

    // Attempt to upload the last file input, if any
    public retry(): Promise<FileData> | null {
        if (this.bufferedFile) {
            this.reset();
            return this.uploadFile(this.bufferedFile);
        }
        return null;
    }

    // Upload file to server and return the file allocation token
    private async upload(file: File): Promise<string> {
        const formData = new FormData();
        formData.append("image", file);

        // Not using fetch, because no ProgressEvent support
        this.xhr = new XMLHttpRequest();
        this.xhr.open("POST", "/api/upload");
        this.xhr.upload.onprogress = e =>
            this.renderProgress(e);
        this.xhr.onabort = () =>
            this.reset();
        this.xhr.send(formData);
        await load(this.xhr);

        if (!this.isUploading) { // Cancelled while uploading
            return "";
        }
        this.isUploading = false;
        const text = this.xhr.responseText;
        if (this.handleResponse(this.xhr.status, text)) {
            this.xhr = null;
            this.button.hidden = true;
            return text;
        }
        return "";
    }

    // Cancel any ongoing upload
    public cancel() {
        if (this.xhr) {
            this.xhr.abort();
            this.xhr = null;
        }
    }

    // Cancel any current uploads and reset form
    // status: status text to use
    public reset(status: string = lang.ui["uploadFile"]) {
        this.isUploading = false;
        this.cancel();
        this.displayStatus(status);
        this.spoiler.hidden = false;
        this.button.hidden = false;
    }

    // Hide the checkbox used to toggle spoilering the image
    public hideSpoilerToggle() {
        this.spoiler.hidden = true;
    }

    // Hide the upload and cancellation button
    public hideButton() {
        this.button.hidden = true;
    }

    // Render client-side upload progress
    private renderProgress({ total, loaded }: LoadProgress) {
        let s: string
        if (loaded === total) {
            s = lang.ui["thumbnailing"]
        } else {
            const n = Math.floor(loaded / total * 100)
            s = `${n}% ${lang.ui["uploadProgress"]}`
        }
        this.displayStatus(s, lang.ui["clickToCancel"]);
    }
}

// Precompute 00 - ff strings for conversion to hexadecimal strings
const precomputedHex = new Array(256);
for (let i = 0; i < 256; i++) {
    precomputedHex[i] = (i < 16 ? '0' : '') + i.toString(16);
}

// Encodes an ArrayBuffer to a hex string
function bufferToHex(buf: ArrayBuffer): string {
    const b = new Uint8Array(buf),
        res = new Array(buf.byteLength);
    for (let i = 0; i < res.length; i++) {
        res[i] = precomputedHex[b[i]];
    }
    return res.join('');
}
