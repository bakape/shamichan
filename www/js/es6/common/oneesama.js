'use strict';System.register([],function(_export,_context){return {setters:[],execute:function(){const _=require('underscore');const index=require('./index');const util=require('./util');var _imports=imports;const config=_imports.config;const pad=util.pad;const parseHTML=util.parseHTML;const break_re=new RegExp("(\\S{"+index.WORD_LENGTH_LIMIT+"})");class OneeSama{readableUTCTime(d,seconds){let html=pad(d.getUTCDate())+' '+this.lang.year[d.getUTCMonth()]+' '+d.getUTCFullYear()+`(${ this.lang.week[d.getUTCDay()] })`+`${ pad(d.getUTCHours()) }:${ pad(d.getUTCMinutes()) }`;if(seconds)html+=`:${ pad(d.getUTCSeconds()) }`;html+=' UTC';return html;}expansionLinks(num){return parseHTML`<span class="act expansionLinks">
				<a href="${ num }" class="history">
					${ this.lang.expand }
				</a>
				] [
				<a href="${ num }?last=${ this.lastN }" class="history">
					${ this.lang.last } ${ this.lastN }
				</a>
			</span>`;}asideLink(inner,href,cls,innerCls){return parseHTML`<aside class="act glass ${ cls }">
				<a ${ href&&`href="${ href }"` }
					${ innerCls&&` class="${ innerCls }"` }
				>
					${ this.lang[inner]||inner }
				</a>
			</aside>`;}replyBox(){return this.asideLink('reply',null,'posting');}newThreadBox(){return this.asideLink('newThread',null,'posting');}}module.exports=OneeSama;}};});
//# sourceMappingURL=../maps/common/oneesama.js.map
