/*
 * Houses both the actual options controler and the options panel renderring
 * logic
 */

let main = require('./main'),
	{$, _, Backbone, state} = main;

// Try to get options from local storage
var options;
try {
	options = JSON.parse(localStorage.options);
}
catch(e) {}
if (!options)
	options = {};
options = module.exports = new Backbone.Model(options);

var OptionsCollection = Backbone.Collection.extend({
	persist() {
		var opts = {};
		this.forEach(function(model) {
			const val = model.getValue();
			if (val === model.get('default'))
				return;
			opts[model.get('id')] = val;
		});
		localStorage.options = JSON.stringify(opts);
	}
});

var optionsCollection = new OptionsCollection();

// Controller template for each individual option
var OptionModel = Backbone.Model.extend({
	initialize(opts) {
		// Condition for loading option. Optional.
		if (opts.load !== undefined && !opts.load)
			return;

		// No type = checkbox + default false
		if (!opts.type)
			this.set('type', 'checkbox');

		const val = this.getValue();
		this.setValue(val);
		if (opts.exec !== undefined) {
			this.listenTo(options, 'change:' + opts.id, this.execListen);
			// Execute with current value
			if (opts.execOnStart !== false)
				opts.exec(val);
		}
		optionsCollection.add(this);
	},
	// Set the option, taking into acount board specifics
	setValue(val) {
		options.set(this.get('id'), val);
	},
	// Return default, if unset
	getValue() {
		const val = options.get(this.get('id'));
		return val === undefined ? this.get('default') : val;
	},
	validate(val) {
		const valid = this.get('validation');
		return valid ? valid(val) : true;
	},
	// Exec wrapper for listening events
	execListen(model, val) {
		this.get('exec')(val);
	}
});

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
	initialize() {
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
for (let spec of require('../common/options')(main.isMobile)) {
	new OptionModel(spec);
}

main.defer(function() {
	new OptionsView({
		el: document.getElementById('options-panel')
	});
});
