'use strict';System.register(['../../util','./header','./image','./etc','./body'],function(_export,_context){var parseHTML,renderHeader,renderImage,renderBanned,renderBacklinks,renderBody;function renderPost(data){const mod=data.mod;const body=data.body;const backlinks=data.backlinks;const banned=data.banned;return parseHTML`${ renderHeader(data) }
		${ renderImage(data) }
		<div class="container">
			${ mod?renderModInfo(mod):'' }
			<blockquote>
				${ renderBody(body) }
			</blockquote>
			<small>
				${ renderBacklinks(backlinks) }
			</small>
			${ banned?renderBanned():'' }
		</div>`;}return {setters:[function(_util){parseHTML=_util.parseHTML;},function(_header){renderHeader=_header.renderHeader;},function(_image){renderImage=_image.renderImage;},function(_etc){renderBanned=_etc.renderBanned;renderBacklinks=_etc.renderBacklinks;},function(_body){renderBody=_body.renderBody;}],execute:function(){function renderSection(data){let cls=arguments.length<=1||arguments[1]===undefined?'':arguments[1];if(data.locked){cls+=' locked';}if(data.editing){cls+=' editing';}data.largeThumb=true;return parseHTML`<section id="p${ data.num }" class="${ cls }">
			<div class="background glass">
				${ renderPost(data) }
				<span class="omit"></span>
			</div>
		</section>`;}_export('renderSection',renderSection);function renderArticle(data){let cls='glass';if(data.editing){cls+=' editing';}return parseHTML`<article id="p${ data.num }" class="${ cls }">
			${ renderPost(data) }
		</article>`;}_export('renderArticle',renderArticle);}};});
//# sourceMappingURL=../../maps/posts/render/posts.js.map
