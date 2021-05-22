import { Post } from "./post"

import { importTemplate } from "../client/util/render"
import { ModerationAction, fileTypes } from "../client/common"
import lang from "../client/lang"
import { secondsToTime } from "../client/util/time"
import countries from "../client/posts/countries"


const modLevelStrings = ["", "janitor", "moderators", "owners", "admin"];

// Base post view class
export default class PostView {
    public el: HTMLElement;
    public model: Post;

    constructor(model: Post) {
        this.model = model;
        
        this.el = document.createElement("article");
        if (model.editing) {
            this.el.classList.add("editing");
        }
        if (model.isDeleted()) {
            this.el.classList.add("deleted");
        }
        this.el.id = `p${model.id}`;
        
        this.model.view = this; // TODO: do not like
        this.el.append(importTemplate("article"));
        this.render();
    }

    private render() {
        this.el.querySelector("blockquote").innerHTML = this.model.body;
        if (this.model.moderation && this.model.moderation.length) {
            this.renderModerationLog();
        }
        this.renderHeader();
        if (this.model.image) {
            this.renderImage();
        }
    }

    // Render the header on top of the post
    private renderHeader() {
        this.renderTime();
        this.renderName();

        const nav = this.el.querySelector("nav"),
            quote = nav.lastElementChild as HTMLAnchorElement,
            { id, flag } = this.model;
        quote.textContent = id.toString();

        // Render country flag, if any
        if (flag) {
            const el = this.el.querySelector(".flag") as HTMLImageElement;
            el.src = `/assets/flags/${flag}.svg`;
            el.title = countries[flag] || flag;
            el.hidden = false;
            if (flag.includes("us-")) {
                const sec = el.cloneNode(true) as HTMLImageElement;
                sec.src = `/assets/flags/us.svg`;
                sec.title = countries["us"] || "us";
                el.insertAdjacentElement("beforebegin", sec);
            }
        }
    }

    // Renders classic absolute timestamp
    private renderTime() {
        const el = this.el.querySelector("time");
        const d = new Date(this.model.time * 1000);
        el.textContent = `${pad(d.getDate())} ${lang.time.calendar[d.getMonth()]} `
            + `${d.getFullYear()} (${lang.time.week[d.getDay()]}) `
            + `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
    }

    // Render the name and tripcode in the header
    private renderName() {
        const el = this.el.querySelector(".name");
        let html = "";
        const { trip, name, auth, sage, id } = this.model;
        if (name || !trip) {
            html += `<span>${name ? escape(name) : lang.posts["anon"]}</span>`;
        }
        if (trip) {
            html += `<code>!${escape(trip)}</code>`;
        }
        if (auth) { // Render staff title
            el.classList.add("admin");
            html += 
                `<span>## ${lang.posts[modLevelStrings[auth]] || "??"}</span>`;
        }
        el.classList.toggle("sage", !!sage);
        el.innerHTML = html;
    }

    private renderModerationLog() {
        for (let { type, length, by, data } of this.model.moderation) {
            let s: string;
            switch (type) {
                case ModerationAction.banPost:
                    s = this.format("banned", by, secondsToTime(length)
                        .toUpperCase(), data);
                    break;
                case ModerationAction.shadowBinPost:
                    s = this.format("shadowBinned", by, secondsToTime(length)
                        .toUpperCase(), data);
                    break;
                case ModerationAction.deletePost:
                    s = this.format("deleted", by);
                    break;
                case ModerationAction.deleteImage:
                    s = this.format("imageDeleted", by);
                    break;
                case ModerationAction.spoilerImage:
                    s = this.format("imageSpoilered", by);
                    break;
                case ModerationAction.lockThread:
                    s = this.format(
                        "threadLockToggled",
                        lang.posts[data === "true" ? "locked" : "unlocked"],
                        by
                    );
                    break;
                case ModerationAction.meidoVision:
                    s = this.format("viewedSameIP", by);
                    break;
                case ModerationAction.purgePost:
                    s = this.format("purgedPost", by);
                    break;
                case ModerationAction.unbanPost:
                    s = this.format("unbanned", by);
                    break;
                case ModerationAction.redirectIP:
                    s = this.format("redirectIP", data, by);
                    break;
                case ModerationAction.redirectThread:
                    s = this.format("redirectThread", data, by);
                    break;
                default:
                    continue;
            }
            const el = document.createElement("b");
            el.classList.add("admin", "post-moderation");
            el.append(s, document.createElement("br"));
            this.el.querySelector(".post-container").append(el);
        }
    }

    // Render the figure and figcaption of a post
    private renderImage() {
        this.el.classList.add("media");

        const el = importTemplate("figure").firstChild as HTMLElement;
        this.el.querySelector(".post-container").prepend(el);
        (el.firstElementChild as HTMLElement).hidden = false;
        this.renderThumbnail();
        this.renderFigCaption();
    }

    // Render the actual thumbnail image
    private renderThumbnail() {
        const el = this.el.querySelector("figure a"),
        { sha1, file_type: file_type, thumb_type: thumbType, dims, spoiler } = this
            .model
            .image,
        src = sourcePath(sha1, file_type);
        let thumb: string,
            [, , thumbWidth, thumbHeight] = dims;
        
        if (thumbType === fileTypes.noFile) {
            // No thumbnail exists
            let file: string;
            switch (file_type) {
                case fileTypes.webm:
                case fileTypes.mp4:
                case fileTypes.mp3:
                case fileTypes.ogg:
                case fileTypes.flac:
                    file = "audio";
                    break;
                default:
                    file = "file";
            }
            thumb = `/assets/${file}.png`;
            thumbHeight = thumbWidth = 150;
        } else if (spoiler) {
            thumb = "/assets/spoil/default.jpg";
            thumbHeight = thumbWidth = 150;
        } else {
            thumb = thumbPath(sha1, thumbType);
        }

        (el.firstElementChild as HTMLImageElement).src = thumb;
        (el.firstElementChild as HTMLImageElement).width = thumbWidth;
        (el.firstElementChild as HTMLImageElement).height = thumbHeight;
        // Remove any existing classes
        (el.firstElementChild as HTMLImageElement).className = "";
    }

    // Render the information caption above the image
    private renderFigCaption() {
        const el = importTemplate("figcaption").firstChild as HTMLElement;
        this.el.querySelector("header").after(el);

        const [info, ...tmp] = Array.from(el.children) as HTMLElement[];
        let link = tmp[tmp.length - 1];

        const data = this.model.image;
        const arr = [];

        if (data.audio) {
            arr.push("♫");
        }

        if (data.length) {
            let s: string;
            if (data.length < 60) {
                s = `0:${pad(data.length)}`;
            } else {
                const min = Math.floor(data.length / 60);
                s = `${pad(min)}:${pad(data.length - min * 60)}`;
            }
            arr.push(s);
        }

        const { size } = data;
        let s: string;
        if (size < (1 << 10)) {
            s = size + " B";
        } else if (size < (1 << 20)) {
            s = Math.round(size / (1 << 10)) + " KB";
        } else {
            const text = Math.round(size / (1 << 20) * 10).toString();
            s = `${text.slice(0, -1)}.${text.slice(-1)} MB`;
        }
        arr.push(s);

        const [w, h] = data.dims;
        if (w || h) {
            arr.push(`${w}x${h}`);
        }

        if (data.artist) {
            arr.push(data.artist);
        }
        if (data.title) {
            arr.push(data.title);
        }

        let html = "";
        for (let s of arr) {
            html += `<span>${escape(s)}</span>`;
        }
        info.innerHTML = html;

        // Render name of an image
        const ext = fileTypes[data.file_type],
            name = `${escape(data.name)}.${ext}`;
        link.innerHTML = name;

        el.hidden = false;
    }


    // C-like sprintf() but only for `%s` tags
    private format(formatKey: string, ...args: string[]): string {
        let i = 0;
        return lang.format[formatKey].replace(/%s/g, _ =>
            args[i++]);
    }
}

// TODO: config.imageRootOverrride
function imageRoot(): string {
    return "/assets/images";
}

// Get the thumbnail path of an image, accounting for thumbnail of specific
// type not being present
function thumbPath(sha1: string, thumbType: fileTypes): string {
    return `${imageRoot()}/thumb/${sha1}.${fileTypes[thumbType]}`;
}

// Resolve the path to the source file of an upload
function sourcePath(sha1: string, fileType: fileTypes): string {
    return `${imageRoot()}/src/${sha1}.${fileTypes[fileType]}`;
}

// Pad an integer with a leading zero, if below 10
// NOTE: utils/index problem
function pad(n: number): string {
	return (n < 10 ? '0' : '') + n
}

const escapeMap: { [key: string]: string } = {
	"&": "&amp;",
	"'": "&#39;", // "&#39;" is shorter than "&apos
	"<": "&lt;",
	">": "&gt;",
	"\"": "&#34;", // "&#34;" is shorter than "&quot;"
}

// Escape a user-submitted unsafe string to protect against XSS.
// NOTE: utils/index
function escape(str: string): string {
	return str.replace(/[&'<>"]/g, char =>
		escapeMap[char])
}