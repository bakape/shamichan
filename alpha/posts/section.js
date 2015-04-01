/*
 * OP and thread related logic
 */

var $ = require('jquery'),
	_ = require('underscore'),
	Backbone = require('backbone'),
	imager = require('./imager'),
	postCommon = require('./common');

var Section = module.exports = Backbone.View.extend({
	tagName: 'section',

	initialize: function () {
		this.listenTo(this.model, {
			'change:locked': this.renderLocked,
			destroy: this.remove,
		});
		this.listenToOnce(this.model, {
			'add': this.renderRelativeTime
		});
		this.initCommon();
	},

	renderHide: function (model, hide) {
		this.$el.next('hr.sectionHr').andSelf().toggle(!hide);
	},

	renderLocked: function (model, locked) {
		this.$el.toggleClass('locked', !!locked);
	},

	remove: function () {
		var replies = this.model.get('replies');
		replies.each(function (post) {
			clear_post_links(post, replies);
		});
		replies.reset();

		this.$el.next('hr.sectionHr').andSelf().remove();
		// Remove from all Posts collection
		Posts.remove(this.model);
		this.stopListening();
	},
});

// Extend with common mixins
_.extend(Section.prototype, imager.Hidamari, postCommon);