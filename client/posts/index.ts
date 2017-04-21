export { Post } from "./model"
export { default as PostView } from "./view"
export { postEvent, postSM, postState, FormModel, identity } from "./posting"
export { default as ImageHandler, toggleExpandAll, setExpandAll } from "./images"
export { clearHidden } from "./hide"
export { renderTime, thumbPath } from "./render"
export { default as PostCollection } from "./collection"
export { findSyncwatches } from "./syncwatch"
export { inlinedPosts } from "./inlineExpansion"

import initEtc from "./etc"
import initPosting from "./posting"
import initMenu from "./menu"
import initInlineExpansion from "./inlineExpansion"
import initHover from "./hover"

export default () => {
	initEtc()
	initPosting()
	initMenu()
	initInlineExpansion()
	initHover()
}

