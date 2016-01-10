'use strict';System.register(['../vendor/dom4','../vendor/js-cookie','./util','./defer','lang','./state','./options/view'],function(_export,_context){var dom4,Cookie,parseEl,parseHTML,defer,execDeferred,lang,state,OptionsPanel;return {setters:[function(_vendorDom){dom4=_vendorDom;},function(_vendorJsCookie){Cookie=_vendorJsCookie;},function(_util){parseEl=_util.parseEl;parseHTML=_util.parseHTML;},function(_defer){defer=_defer.defer;execDeferred=_defer.execDeferred;},function(_lang){lang=_lang.default;},function(_state){state=_state;},function(_optionsView){OptionsPanel=_optionsView.default;}],execute:function(){defer(() => new OptionsPanel());const cookieVersion=3;if(localStorage.cookieVersion!=cookieVersion){for(let cookie in Cookie.get()){const paths=config.boards.enabled.slice();paths.push('','/');for(let path of paths){Cookie.remove(cookie,{path});}}localStorage.cookieVersion=cookieVersion;}if(/[&\?]debug=true/.test(location.href)){config.debug=true;}document.head.appendChild(parseEl(parseHTML`<style>
		.locked:after {
			content: "${ lang.thread_locked }";
		}
		.locked > header nav:after {
			content: " (${ lang.locked })";
		}
	</style>`));execDeferred();}};});
//# sourceMappingURL=maps/main.js.map
