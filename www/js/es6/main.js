'use strict';System.register(['../vendor/dom4','../vendor/js-cookie','./util'],function(_export){var dom4,Cookie,parseEl,parseHTML;return {setters:[function(_vendorDom){dom4=_vendorDom;},function(_vendorJsCookie){Cookie=_vendorJsCookie;},function(_util){parseEl=_util.parseEl;parseHTML=_util.parseHTML;}],execute:function(){const deferred=[];function defer(func){deferred.push(func);}_export('defer',defer);function execDeferred(){while(deferred.length>0){deferred.shift()();}}_export('execDeferred',execDeferred);const config=window.config;_export('config',config);config.mediaURL=config.hard.HTTP.media;const configHash=window.configHash;_export('configHash',configHash);const clientHash=window.clientHash;_export('clientHash',clientHash);const isMobile=window.isMobile;_export('isMobile',isMobile);const $threads=document.query('threads');_export('$threads',$threads);const $name=document.query('#name');_export('$name',$name);const $email=document.query('#email');_export('$email',$email);const $banner=document.query('#banner');_export('$banner',$banner);const cookieVersion=3;if(localStorage.cookieVersion!=cookieVersion){for(let cookie in Cookie.get()){const paths=config.boards.enabled.slice();paths.push('','/');for(let path of paths){Cookie.remove(cookie,{path});}}localStorage.cookieVersion=cookieVersion;}if(/[&\?]debug=true/.test(location.href)){config.hard.debug=true;}document.head.appendChild(parseEl(parseHTML`<style>
		.locked:after {
			content: "${ lang.thread_locked }";
		}
		.locked > header nav:after {
			content: " (${ lang.locked })";
		}
	</style>`));execDeferred();}};});
//# sourceMappingURL=maps/main.js.map
