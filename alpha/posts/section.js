/*
 * OP and thread related logic
 */

var $ = require('jquery'),
	_ = require('underscore'),
	Backbone = require('backbone'),
	imager = require('./imager'),
	main = require('../main'),
	postCommon = require('./common'),
	state = require('../state');

var Section = module.exports = Backbone.View.extend({
	tagName: 'section',

	initialize: function () {
		// On the live page only
		if (this.$el.is(':empty') && state.page.get('page') === -1)
			this.render();
		this.listenTo(this.model, {
			'change:locked': this.renderLocked,
			destroy: this.remove,
		});
		this.listenToOnce(this.model, {
			'add': this.renderRelativeTime
		});
		this.initCommon();
	},

	render: function() {
		main.oneeSama.links = this.mode.get('links');
		this.setElement(main.oneeSama.monomono(this.model.attributes).join(''));
		this.$el.insertAfter(main.$threads.children('hr.sectionHr').first());
		this.$el.after('<hr class="sectionHr"/>');
		return this;
	},

	renderHide: function (model, hide) {
		this.$el.next('hr.sectionHr').andSelf().toggle(!hide);
	},

	renderLocked: function (model, locked) {
		this.$el.toggleClass('locked', !!locked);
	},

	remove: function () {
		this.$el.next('hr.sectionHr').andSelf().remove();
		this.stopListening();
		return this;
	},
});

// Extend with common mixins
_.extend(Section.prototype, imager.Hidamari, postCommon);