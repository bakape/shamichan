/*
 * Houses both the actual options controler and the options panel renderring
 * logic
 */

import {extend, each, find} from 'underscore'
import * as Backbone from 'backbone'
import {defer} from 'main'
import ModalView from '../modal'
import {once} from '../util'
import opts from './opts'
import render from './render'

// Delete legacy options localStorage entry, if any
localStorage.removeItem("options")
const options = new Backbone.Model()
export default options

// Create and option model for each object in the array
for (let spec of opts) {
	new OptionModel(spec)
}

// Expensive comutation and not emediatly needed. Put off till later.
defer(() => new OptionsView())
