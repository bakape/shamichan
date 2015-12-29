/*
 * Houses both the actual options controler and the options panel renderring
 * logic
 */

import {_, Backbone, state, defer} from 'main'
import opts from './opts'

// Try to get options from local storage
let options
try {
	options = JSON.parse(localStorage.options)
}
catch(e) {}
if (!options) {
	options = {}
}
export default options = new Backbone.Model(options)

const optionModels = {}

/**
 * Coontroler for each individual option
 */
class OptionModel {
	/**
	 * Create new option model from template model
	 * @param {Object} model
	 */
    constructor(model) {
		// Condition for loading option. Optional.
		if (model.load !== undefined && !model.load) {
			return
		}
		_.extend(this, model)

		// No type = checkbox + default false
		if (!this.type) {
		    this.type = 'checkbox'
		}

		// Store option value in central stotage options Backbone model
		const val = options.attributes[this.id] = this.get()
		options.on('change:' + this.id, (options, val) =>
			this.onchange(val))
		if (this.execOnStart !== false) {
		    this.execute(this.val)
		}
		optionModels[this.id] = this
    }

	/**
	 * Read value from localStorage
	 * @returns {string}
	 */
	read() {
	    return localStorage.getItem(this.id)
	}

	/**
	 * Retrieve option value from storage and parse result. If none, return
	 * default.
	 * @returns {string|bool|int}
	 */
	get() {
		const stored = this.read()
	    if (!stored) {
	        return this.default
	    } else {
			if (stored === 'false') {
		        return false
		    }
			if (stored === "true") {
		        return true
		    }
			const num = parseInt(stored, 10)
			if (num || num === 0) {
			    return num
			}
			return this.default
		}
	}

	/**
	 * Handler to be executed on field change in central options storage model
	 * @param {*} val
	 */
	onChange(val) {
	    this.execute(val)
		this.set(val)
	}

	/**
	 * Execute handler function, if any
	 * @param {*} val
	 */
	execute(val) {
	    if (this.exec) {
	        this.exec(val)
	    }
	}

	/**
	 * Write value to localStorage, if needed
	 * @param {*} val
	 */
	set(val) {
	    if (this.validate(val) && val !== this.default || this.read()) {
	        localStorage.setItem(this.id,val)
	    }
	}

	/**
	 * Perform value validation, if any. Othervise return true.
	 * @param {*} val
	 * @returns {bool}
	 */
	validate(val) {
	    if (this.validation) {
	        return this.validation(val)
	    }
		return true
	}
}

// Highlight options button by fading out and in, if no options are set
(function() {
	if (localStorage.getItem('options')) {
		return
	}
	const el = document.query('#options')

	function fadeOutAndIn(el) {
		el.style.opacity = 1
		let out = true,
			clicked
		tick()

		function tick() {
			// Stop
			if (clicked) {
			    el.style.opacity = 1
				return
			}

	    	el.style.opacity = +el.style.opacity + (out ? -0.01 : 0.01)
			const now = +el.style.opacity

			// Reverse direction
			if ((out && now <= 0) || (!out && now >= 1)) {
			    out = !out
			}
			requestAnimationFrame(tick)
		}

		el.addEventListener("click", () => clicked = true)
	}
})()

// View of the options panel
var OptionsView = Backbone.View.extend({
	initialize() {
		// Render the options panel
		this.setElement(util.parseEl(require('./render')()))
		document.body.append(this.el)

		// Set the options in the panel to their appropriate values
		optionsCollection.each(model => {
			let $el = this.$el.find('#' + model.get('id'));
			/*
			 * No corresponding element in panel. Can be caused by config
			 * mismatches.
			 */
			if (!$el.length)
				return;
			const type = model.get('type'),
				val = model.getValue();
			if (type == 'checkbox')
				$el.prop('checked', val);
			else if (type == 'number' || type instanceof Array)
				$el.val(val);
			else if (type == 'shortcut')
				$el.val(String.fromCharCode(val).toUpperCase());
			// 'image' type simply falls through, as those don't need to be set
		});
		this.$hidden = this.$el.find('#hidden');
		main.reply('hide:render', this.renderHidden, this);
	},
	events: {
		'click .option_tab_sel>li>a': 'switchTab',
		'change': 'applyChange',
		'click #export': 'export',
		'click #import': 'import',
		'click #hidden': 'clearHidden'
	},
	switchTab(event) {
		event.preventDefault();
		var $a = $(event.target);
		// Unhighight all tabs
		this.$el.children('.option_tab_sel').find('a').removeClass('tab_sel');
		// Hightlight the new one
		$a.addClass('tab_sel');
		// Switch tabs
		var $li = this.$el.children('.option_tab_cont').children('li');
		$li.removeClass('tab_sel');
		$li.filter('.' + $a.data('content')).addClass('tab_sel');
	},
	// Propagate options panel changes to the models and localStorage
	applyChange(event) {
		var $target = $(event.target),
			model = optionsCollection.get($target.attr('id')),
			val;
		if (!model)
			return;
		const type = model.get('type');
		if (type == 'checkbox')
			val = $target.prop('checked');
		else if (type == 'number')
			val = parseInt($target.val());
		// Not recorded; extracted directly by the background handler
		else if (type == 'image')
			return main.request('background:store', event.target);
		else if (type == 'shortcut')
			val = $target.val().toUpperCase().charCodeAt(0);
		else
			val = $target.val();

		if (!model.validate(val))
			return $target.val('');
		model.setValue(val);
		optionsCollection.persist();
	},
	// Dump options to file
	export() {
		var a = document.getElementById('export')
		a.setAttribute('href', window.URL
			.createObjectURL(new Blob([JSON.stringify(localStorage)], {
				type: 'octet/stream'
			}))
		);
		a.setAttribute('download', 'meguca-config.json');
	},
	// Import options from file
	import(event) {
		// Proxy to hidden file input
		event.preventDefault();
		var $input = this.$el.find('#importSettings');
		$input.click();
		$input.one('change', function() {
			var reader = new FileReader();
			reader.readAsText($input[0].files[0]);
			reader.onload = function(e) {
				var json;
				// In case of curruption
				try {
					json = JSON.parse(e.target.result);
				}
				catch(e) {
					alert('Import failed. File corrupt');
				}
				if (!json)
					return;
				localStorage.clear();
				for (let key in json) {
					localStorage[key] = json[key];
				}
				alert('Import successfull. The page will now reload.');
				location.reload(true);
			};
		});
	},
	// Hiden posts counter and reset link
	renderHidden(count) {
		let $el = this.$hidden;
		$el.text($el.text().replace(/\d+$/, count));
	},
	clearHidden() {
		main.request('hide:clear');
		this.renderHidden(0);
	}
});

// Create and option model for each object in the array
for (let spec of opts) {
	new OptionModel(spec)
}

let optionsPanel
defer(() => optionsPanel = new OptionsView())
