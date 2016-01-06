'use strict';System.register(['backbone','main'],function(_export){var View,defer;return {setters:[function(_backbone){View=_backbone.View;},function(_main){defer=_main.defer;}],execute:function(){_export('default',View.extend({className:'modal bmodal glass',initialize(){defer(() => {this.render();document.body.append(this.el);});}}));}};});
//# sourceMappingURL=maps/modal.js.map
