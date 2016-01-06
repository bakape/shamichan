'use strict';System.register(['../util','../header'],function(_export){var parseHTML,renderHeader;function renderPost(data){const image=data.image;const mod=data.mod;const body=data.body;const backlinks=data.backlinks;const banned=data.banned;return parseHTML`${ renderHeader(data) }
        ${ renderImage(image) }
        <div class="container">
            ${ renderModInfo(mod) }
            <blockquote>
                ${ renderBody(body) }
            </blockquote>
            <small>
                ${ backlinks?renderBacklinks(backlinks):'' }
            </small>
            ${ banned?renderBanned():'' }
        </div>`;}return {setters:[function(_util){parseHTML=_util.parseHTML;},function(_header){renderHeader=_header.renderHeader;}],execute:function(){function renderSection(data,cls=''){if(data.locked){cls+=' locked';}if(data.editing){cls+=' editing';}data.image.large=true;return parseHTML`<section id="p${ data.num }" class="${ cls }">
            <div class="background glass">
                ${ renderPost(data) }
                <span class="omit"></span>
            </div>
        </section>`;}_export('renderSection',renderSection);function renderArticle(data){let cls='glass';if(data.editing){cls+=' editing';}return parseHTML`<article id="p${ data.num }" class="${ cls }">
            ${ renderPost(data) }
        </article>`;}_export('renderArticle',renderArticle);}};});
//# sourceMappingURL=../../maps/posts/render/index.js.map
