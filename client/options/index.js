/*
 * Houses both the actual options controler and the options panel renderring
 * logic
 */
'use strict';

var _ = require('underscore'),
	$ = require('jquery'),
	Backbone = require('backbone');

// Try to get options from local storage
var options;
try {
	options = JSON.parse(localStorage.options);
}
catch(e) {
}
if (!options)
	options = {};
options.id = 'options';
options = module.exports = new Backbone.Model(options);

// Require these after the options backbone model has been created
var background = require('../background'),
	banner = require('../banner'),
	main = require('../main'),
	state = require('../state');

var OptionsCollection = Backbone.Collection.extend({
	persist: function() {
		var opts = {};
		this.forEach(function(model) {
			const val = model.getValue();
			if (val === model.get('default'))
				return;
			opts[model.get('storedId')] = val;
		});
		localStorage.options = JSON.stringify(opts);
	}
});

var optionsCollection = new OptionsCollection();

// Controller template for each individual option
var OptionModel = Backbone.Model.extend({
	initialize: function(obj) {
		// Condition for loading option. Optional.
		if (obj.load !== undefined && !obj.load)
			return;
		this.set(obj);
		// No type = checkbox + default false
		if (!obj.type)
			this.set({type: 'checkbox', default: false});
		/*
		 * Some options differ per board. Store the id that will be used in the
		 * options model for searching purposes.
		 */
		var id = obj.id;
		if (obj.boardSpecific)
			id = 'board.' + state.page.get('board') + '.' + id;
		this.set('storedId', id);

		const val = this.getValue();
		options.set(id, val);
		if (obj.exec !== undefined) {
			this.listenTo(options, 'change:' + id, this.execListen);
			// Execute with current value
			if (obj.execOnStart !== false)
				obj.exec(val);
		}
		optionsCollection.add(this);
	},

	// Set the option, taking into acount board specifics
	setStored: function(val) {
		options.set(this.get('storedId'), val);
	},

	// Return default, if unset
	getValue: function() {
		const val = options.get(this.get('storedId'));
		return val === undefined ? this.get('default') : val;
	},

	validate: function(val) {
		const valid = this.get('validation');
		return valid ? valid(val) : true;
	},

	// Exec wrapper for listening events
	execListen: function(model, val) {
		this.get('exec')(val);
	}
});

// Create and option model for each object in the array
const optCommon = require('../../common/options');
for (let i = 0, lim = optCommon.length; i < lim; i++) {
	new OptionModel(optCommon[i]);
}

// Highlight options button, if no options are set
(function() {
	if (localStorage.getItem('options'))
		return;
	var $el = $('#options');
	$el.addClass('noOptions');

	function fadeout() {
		$el.filter('.noOptions').fadeOut(fadein);
	}

	function fadein() {
		$el.fadeIn();
		// Stop animation, if options pannel is opened
		if ($el.filter('.noOptions').length)
			fadeout();
	}

	fadeout();

	$el.click(function() {
		$el.removeClass('noOptions');
	});
})();

// View of the options panel
var OptionsView = Backbone.View.extend({
	initialize: function() {
		this.setElement(document.getElementById('options-panel'));
		// Set the options in the panel to their appropriate values
		optionsCollection.each(function(model) {
			var $el = this.$el.find('#' + model.get('id'));
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
		}, this);
	},

	events: {
		'click .option_tab_sel>li>a': 'switchTab',
		'change': 'applyChange',
		'click #export': 'export',
		'click #import': 'import'
	},

	switchTab: function(event) {
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
	applyChange: function(event) {
		var $target = $(event.target),
			model = optionsCollection.findWhere({
				id: $target.attr('id')
			}),
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
			// FIXME
			return; //background.genCustom(target.result);
		else if (type == 'shortcut')
			val = $target.val().toUpperCase().charCodeAt(0);
		else
			val = $target.val();

		if (!model.validate(val))
			return $target.val('');
		model.setStored(val);
		optionsCollection.persist();
	},

	// Dump options to file
	export: function() {
		var a = document.createElement('a');
		a.setAttribute('href', window.URL
			.createObjectURL(new Blob([JSON.stringify(localStorage)], {
				type: 'octet/stream'
			}))
		);
		a.setAttribute('download', 'meguca-config.json');
		a.click();
	},

	// Import options from file
	import: function(event) {
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
				location.reload();
			};
		});
	}
});

var optionsView;
// Render it after the current stack clears,for a bit more responsiveness
_.defer(function() {
	optionsView = new OptionsView();
});

// TODO: BoardSpecific option unloading code for inter-board push state navigation
