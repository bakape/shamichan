/**
 * Various minor windows and the base view for all modals
 */

import {View} from 'backbone'
import {defer} from 'main'

export default View.extend({
    className: 'modal bmodal glass',

    /**
     * Calls the subview-specific initialization methods
     */
    initialize() {
        defer(() => {
            this.render()
            document.body.append(this.el)
        })
    }

    // TODO: Add close button and unify modal structure

})
