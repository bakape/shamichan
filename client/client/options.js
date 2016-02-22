/**
 * Small container module for easily importing the loaded options.
 */

import Model from './model'

// Delete legacy options localStorage entry, if any
localStorage.removeItem("options")
export default new Model()
