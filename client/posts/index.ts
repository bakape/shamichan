export { Post } from "./model"
export { default as PostView } from "./view"
export { postEvent, postSM, postState, FormModel, identity, expandThreadForm } from "./posting"
export { default as ImageHandler, toggleExpandAll, thumbPath } from "./images"
export { clearHidden, hideRecursively } from "./hide"
export { toggleGallery } from "./filter"
export * from "./render"
export { default as PostCollection } from "./collection"
export { findSyncwatches } from "./syncwatch"

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

