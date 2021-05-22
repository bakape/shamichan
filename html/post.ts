import PostView from "./view"

import {
    PostData, PostLink, ModerationLevel, ModerationEntry, Command, ImageData, ModerationAction
} from "../client/common"

export class Post {
    public view: PostView;

    public editing: boolean;
    public sage: boolean;
    public image: ImageData;
    public time: number;
    public id: number;
    public op: number;
    public body: string;
    public name: string;
    public trip: string;
    public auth: ModerationLevel;
    public board: string;
    public flag: string;
    public links: PostLink[];
    public commands: Command[];
    public moderation: ModerationEntry[];

    constructor(attrs: PostData) {
        extend(this, attrs);
    }

    public isDeleted(): boolean {
        if (!this.moderation) {
            return false;
        }
        for (let { type } of this.moderation) {
            if (type === ModerationAction.deletePost) {
                return true;
            }
        }

        return false;
    }
}

// Copy all properties from the source object to the destination object. Nested
// objects are extended recursively.
// NOTE: importing from "../client/utils/index" imports goddamn everything from
// the "../client" tree
function extend(dest: {}, source: {}) {
	for (let key in source) {
		const val = source[key]
		if (typeof val === "object" && val !== null) {
			const d = dest[key]
			if (d) {
				extend(d, val)
			} else {
				dest[key] = val
			}
		} else {
			dest[key] = val
		}
	}
}