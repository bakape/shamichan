import { Post } from "./post"
import PostView from "./view"
// Have to be very specific with imports
import { fetchJSON } from "../client/util/fetch"
import { emitChanges, ChangeEmitter } from "../client/util/changes"
import { PostData } from "../client/common"

interface MouseMove extends ChangeEmitter {
    event: MouseEvent;
}

const overlay = document.getElementById("hover-overlay");

// Currently displayed preview, if any
let postPreview: PostPreview;

// Centralized mousemove trget tracking
const mouseMove = emitChanges<MouseMove>({
    event: {
        target: null,
    },
} as MouseMove);


class PostPreview {
    public el: HTMLElement;
    private parent: HTMLElement;

    constructor(model: Post, parent: HTMLElement) {
        this.parent = parent;
        this.el = model.view.el;
        this.el.classList.add("preview");
        this.render();
    }

    private render() {
        const fc = overlay.firstChild;
		if (fc !== this.el) {
			if (fc) {
				fc.remove();
			}
			overlay.append(this.el);
		}
        this.position();
    }

    // Position the preview element relative to its parent link
    private position() {
        const rect = this.parent.getBoundingClientRect();

        // The preview will never take up more than 100% screen width, so no
		// need for checking horizontal overflow. Must be applied before
		// reading the height, so it takes into account post resizing to
		// viewport edge
        this.el.style.left = `${rect.left}px`;

        const height = this.el.offsetHeight;
        let top = rect.top - height - 5;

        // If post gets cut off at the top, put it below the link
        if (top < 0) {
            top += height + 23;
        }
        this.el.style.top = `${top}px`;
    }

    // Remove reference to this view from the module
    public remove() {
        postPreview = null;
        this.el.remove();
    }
}

async function renderPostPreview(event: MouseEvent) {
    let target = event.target as HTMLElement;
    if (!target.matches || !target.matches("a.post-link, .hash-link")) {
        return;
    }
    if (target.classList.contains("hash-link")) {
        target = target.previousElementSibling as HTMLElement;
    }
    if (target.matches("em.expanded > a")) {
        return;
    }
    const id = parseInt(target.getAttribute("data-id"));
    if (!id) {
        return;
    }

    const [data] = await fetchJSON<PostData>(`/json/post/${id}`);
    if (!data) {
        return;
    }
    const post = new Post(data);
    new PostView(post); // TODO: I do not like this
    postPreview = new PostPreview(post, target);
}

// Clear any previews
function clear() {
    if (postPreview) {
        postPreview.remove();
        postPreview = null;
    }
}

// Bind mouse movement event listener
function onMouseMove(event: MouseEvent) {
    if (event.target !== mouseMove.event.target) {
        clear();
        mouseMove.event = event;
    }
}

export default () => {
    document.addEventListener("mousemove", onMouseMove, {
        passive: true,
    });
    mouseMove.onChange("event", renderPostPreview);
}