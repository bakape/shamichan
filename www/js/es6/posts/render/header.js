'use strict';System.register(['underscore','../util'],function(_export,_context){var escape,parseHTML,pad;function resolveName(data){let html='';const trip=data.trip;const name=data.name;const auth=data.auth;if(name||!trip){if(name){html+=escape(name);}else {html+=lang.anon;}if(trip){html+=' ';}}if(trip){html+=`<code>${ escape(trip) }</code>`;}if(auth){html+=` ## ${ imports.hotConfig.staff_aliases[auth]||auth }`;}return html;}function readableTime(time){let d=new Date(time);return pad(d.getDate())+' '+lang.year[d.getMonth()]+' '+d.getFullYear()+`(${ lang.week[d.getDay()] })`+`${ pad(d.getHours()) }:${ pad(d.getMinutes()) }`;}function relativeTime(then,now){let time=Math.floor((now-then)/60000),isFuture;if(time<1){if(time>-5){return this.lang.just_now;}else {isFuture=true;time=-time;}}const divide=[60,24,30,12],unit=['minute','hour','day','month'];for(let i=0;i<divide.length;i++){if(time<divide[i]){return lang.ago(time,lang.time[unit[i]],isFuture);}time=Math.floor(time/divide[i]);}return lang.ago(time,lang.time.year,isFuture);}return {setters:[function(_underscore){escape=_underscore.escape;},function(_util){parseHTML=_util.parseHTML;pad=_util.pad;}],execute:function(){function renderHeader(data){const num=data.num;const op=data.op;const subject=data.subject;return parseHTML`<header>
            <input type="checkbox" class="postCheckbox">
            ${ subject?`<h3>「${ escape(data.subject) }」</h3>`:'' }
            ${ renderName(data) }
            ${ renderTime(data.time) }
            <nav>
                <a href="${ this.postURL(num,op) }" class="history">
                    No.
                </a>
                <a href="${ this.postURL(num,op) }" class="quote">
                    ${ num }
                </a>
            </nav>
        </header>
        <span class="oi control" data-glyph="chevron-bottom"></span>`;}_export('renderHeader',renderHeader);function renderName(data){let html='<b class="name';const auth=data.auth;const email=data.email;if(auth){html+=` ${ auth==='admin'?'admin':'moderator' }`;}html+='">';if(email){const attrs={class:'email',href:'mailto:'+encodeURI(email),target:'blank'};html+=parseHTML`<a ${ attrs }>`;}html+=resolveName(data);if(email){html+='</a>';}html+='</b>';if(data.mnemonic){html+=' '+renderMnemonic(data.mnemonic);}return html;}_export('renderName',renderName);function renderMnemonic(mnemonic){return `<b class="mod addr">${ mnem }</b>`;}_export('renderMnemonic',renderMnemonic);function renderTime(time){let title,text;const readable=readableTime(time);if(options.get('relativeTime')){title=readable;text=relativeTime(time,Date.now());}return parseHTML`<time title="${ title }">
            ${ text||readable }
        </time>`;}_export('renderTime',renderTime);}};});
//# sourceMappingURL=../../maps/posts/render/header.js.map
