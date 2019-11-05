import { page, posts } from "../state";
import { isAtBottom, makeEl, HTML } from "../util";
import { toggleLinkReferences } from "./inlineExpansion";
import lang from "../lang";

function renderOmitted(posts: number, images: number) {
	const op = document.querySelector("#thread-container article");
	const old = op.querySelector(".omit");
	if (old) {
		posts += parseInt(old.getAttribute("data-omit"));
		images += parseInt(old.getAttribute("data-image-omit"));
		old.remove();
	}

	let s: string;
	if (!images) {
		s = lang.format["postsOmitted"].replace("%d", posts.toString());
	} else {
		let i = 0;
		s = lang.format["postsAndImagesOmitted"].replace(/\%d/g, () => {
			if (i++) {
				return images.toString();
			}
			return posts.toString();
		});
	}
	const el = makeEl(HTML
		`<span class="omit spaced" data-omit="${posts.toString()}" data-image-omit="${images.toString()}">
			${s}
			<span class="act">
				<a href="${page.thread.toString()}">
					${lang.posts['seeAll']}
				</a>
			</span>
		</span>`);

	op.querySelector(".post-container").after(el);
}

// Attempt to reduce thread size, if last100 enables and at thread bottom
export function lightenThread() {
	if (!page.thread
		|| page.lastN !== 100
		|| !isAtBottom()
		|| posts.size() <= 100
	) {
		return;
	}

	const models = [...posts]
		.filter(m =>
			m.id !== page.thread)
		.sort((a, b) => {
			if (a.time < b.time) {
				return -1;
			}
			if (a.time > b.time) {
				return 1;
			}
			return 0;
		});
	let removedPosts = 0;
	let removedImages = 0;
	while (models.length > 99) { // 100 - OP
		const m = models.shift();
		posts.remove(m);
		removedPosts++;
		if (m.image) {
			removedImages++;
		}

		if (!m.view) {
			continue;
		}
		const el = m.view.el;
		const parent = el.closest("article");
		if (parent) {
			toggleLinkReferences(parent, m.id, false);
		}
		el.remove();
	}

	renderOmitted(removedPosts, removedImages);
}
