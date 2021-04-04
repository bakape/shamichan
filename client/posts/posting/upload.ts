import lang from '../../lang';
import { load, trigger } from '../../util';
import { Post } from "../model";
import { View } from "../../base";
import { config, page } from "../../state";
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

const micSVG = `<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8" viewBox="0 0 8 8">
    <path d="M2.91-.03a1 1 0 0 0-.13.03 1 1 0 0 0-.78 1v2a1 1 0 1 0 2 0v-2a1 1 0 0 0-1.09-1.03zm-2.56 2.03a.5.5 0 0 0-.34.5v.5c0 1.48 1.09 2.69 2.5 2.94v1.06h-.5c-.55 0-1 .45-1 1h4.01c0-.55-.45-1-1-1h-.5v-1.06c1.41-.24 2.5-1.46 2.5-2.94v-.5a.5.5 0 1 0-1 0v.5c0 1.11-.89 2-2 2-1.11 0-2-.89-2-2v-.5a.5.5 0 0 0-.59-.5.5.5 0 0 0-.06 0z"
transform="translate(1)" />
</svg>`;
const stopSVG = `<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8" viewBox="0 0 8 8">
    <path d="M0 0v6h6v-6h-6z" transform="translate(1 1)" />
</svg>`;

declare class MediaRecorder {
    state: string;
    ondataavailable: (e: MessageEvent) => void;
    onpause: () => void;
    onstop: () => void;
    onerror: (e: any) => void;

    constructor(stream: MediaStream);
    start(): void;
    stop(): void;

    static isTypeSupported(mime: string): boolean;
}

// View for handling file uploads
export default class UploadForm extends View<Post> {
    private isUploading: boolean;

    private spoiler: HTMLElement;
    private mask: HTMLElement;
    private button: HTMLElement;

    private micButton: HTMLElement | null;
    private audioChunks: any[] = [];
    private recorder: MediaRecorder | null;

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
        this.mask = el
            .querySelector(`span[data-id="mask"]`) as HTMLInputElement;
        this.button = el.querySelector("button");

        // People who want to hide their filenames will probably want to do it
        // a lot. Store their choice
        this.inputElement("mask").checked =
            (localStorage.getItem("mask") === "true");

        this.mask.addEventListener(
            "click",
            () => {
                localStorage.setItem(
                    "mask",
                    String(this.inputElement("mask").checked)
                );
            },
            { passive: true },
        );
        this.button.addEventListener(
            "click",
            () => {
                if (this.isUploading) {
                    this.reset();
                } else if (this.canAllocImage()) {
                    this.hiddenInput.click();
                }
            },
            { passive: true },
        );
        this.hiddenInput.addEventListener(
            "change",
            () => {
                if (this.canAllocImage() && this.hiddenInput.files.length) {
                    trigger("getPostModel").
                        uploadFile(this.hiddenInput.files[0]);
                }
            },
            { passive: true },
        );

        if (navigator.mediaDevices
            && navigator.mediaDevices.getUserMedia
        ) {
            this.micButton = document.createElement("a");
            this.micButton.classList.add("record-button", "svg-link");
            this.micButton.innerHTML = micSVG;
            el.children[0].after(this.micButton);

            this.micButton.addEventListener(
                "click",
                () => {
                    if (!this.recorder) {
                        navigator
                            .mediaDevices
                            .getUserMedia({
                                audio: true,
                            })
                            .then(stream => {
                                this.recorder = new MediaRecorder(stream);
                                this.recorder.start();
                                this.micButton.innerHTML = stopSVG;

                                this.recorder.ondataavailable = ({ data }) =>
                                    this.audioChunks.push(data);
                                this.recorder.onerror = ({ error }) => {
                                    this.recorder = null;
                                    console.error(error);
                                    alert(error);
                                    this.reset();
                                };
                                this.recorder.onpause = () =>
                                    this.recorder.stop();
                                this.recorder.onstop = () => {
                                    trigger("getPostModel").uploadFile(
                                        new File(
                                            this.audioChunks,
                                            "recording.ogg",
                                        ),
                                    );
                                    this.micButton.hidden = true;
                                    this.recorder = null;
                                    this.audioChunks = [];
                                };
                            });
                    } else {
                        this.recorder.stop();
                    }
                }
            )
        }
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

        let uploadName = file.name;
        let token: string;
        // First send an sha1 hash to the server, in case it already has
        // the file thumbnailed and we don't need to upload.
        const sha1 = await getSHA(file);
        if (sha1 !== "") {
            if (this.inputElement("mask").checked) {
                uploadName = sha1;
            }
            const res = await fetch("/api/upload-hash", {
                method: "POST",
                body: sha1,
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
            name: uploadName,
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
    public async retry(): Promise<FileData | null> {
        if (this.bufferedFile) {
            this.reset();
            return await this.uploadFile(this.bufferedFile);
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
            this.mask.hidden = true;
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
        this.mask.hidden = false;
        this.button.hidden = false;

        this.audioChunks = [];
        if (this.micButton) {
            this.micButton.hidden = false;
            this.micButton.innerHTML = micSVG;
        }
    }

    // Hide the checkbox used to toggle spoilering the image
    public hideSpoilerToggle() {
        this.spoiler.hidden = true;
    }

    // Hide the checkbox used to toggle masking the filename
    public hideMaskToggle() {
        this.mask.hidden = true;
    }

    // Hide the upload and cancellation button
    public hideButton() {
        this.button.hidden = true;
        if (this.micButton) {
            this.micButton.hidden = true;
        }
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

// Return the hex-encoded SHA-1 hash of a file, or an empty string
// if the crypto API is unavailable
async function getSHA(file: File): Promise<string> {
    // Detect, if the crypto API can be used
    if (location.protocol === "https:" || location.hostname === "localhost") {
        const r = new FileReader();
        r.readAsArrayBuffer(file);
        const { target: { result } }
            = await load(r) as ArrayBufferLoadEvent;
        return bufferToHex(await crypto.subtle.digest("SHA-1", result));
    }
    return "";
}

// Mask input file's filename
async function maskFile(input: HTMLInputElement) {
    if (input.files.length === 0) {
        return;
    }
    // File.name is immutable, replace contents of input element
    // with cloned+renamed file
    const blob = input.files[0];
    const name = await getSHA(blob);
    if (name === "") {
        return;
    }
    const newfile = new File([blob], name, {type: blob.type});
    const data  = new DataTransfer();
    data.items.add(newfile);
    input.files = data.files;
}

export function initUpload() {
    // Add event to optionally mask uploaded file's name when creating
    // a new thread
    if (!page.thread) {
        const form = document.
            getElementById("new-thread-form") as HTMLFormElement;
        const mask = form.
            querySelector("input[name=mask]") as HTMLInputElement;

        form.addEventListener("submit", async (e) => {
            e.preventDefault();
            if (mask.checked) {
                await maskFile(
                    form.querySelector("input[type=file]") as HTMLInputElement
                );
            }
            // Doesn't trigger submit event
            form.submit();
        });
    }
}
