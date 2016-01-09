'use strict';System.register([],function(_export,_context){return {setters:[],execute:function(){const main=require('./main');const Backbone=main.Backbone;const common=main.common;const dispatcher=main.dispatcher;const etc=main.etc;const options=main.options;const modalMap={'options':'options-panel','banner_identity':'identity','banner_FAQ':'FAQ','banner_schedule':'schedule'};const BannerView=Backbone.View.extend({initialize(){this.center=document.getElementById('banner_center');this.info=document.getElementById('banner_info');main.reply('banner:radio:clear',this.clearRadio,this);},events:{'click .bfloat':'revealBmodal'},renderInfo(msg){this.info.innerHTML=msg;},revealBmodal(event){const bmodal=modalMap[event.target.closest('.bfloat').getAttribute('id')];if(!bmodal)return;const el=document.getElementById(bmodal),isShown=el&&getComputedStyle(el).display!=='none';for(let el of document.queryAll('.bmodal')){el.style.display='none';}if(isShown)return;el.style.top=document.getElementById('banner').offsetHeight+5+'px';el.style.display='block';},renderRadio(data){data=JSON.parse(data);const attrs={title:main.lang.googleSong,href:`https://google.com/search?q=${ encodeURIComponent(data.np) }`,target:'_blank'};this.center.innerHTML=common.parseHTML`<a href="http://r-a-d.io/" target="_blank">
				[${ data.listeners }] ${ data.dj }
			</a>
			&nbsp;&nbsp;
			<a ${ attrs }>
				<b>${ data.np }</b>
			</a>`;},clearRadio(){this.center.innerHTML='';}});const banner=exports.view=new BannerView({el:document.getElementById('banner')});const NotificationView=exports.notification=Backbone.View.extend({initialize(msg){this.render(msg);},events:{'click':'remove'},render(msg){for(let el of document.queryAll('.notification')){el.remove();}const attrs={class:'notification modal',style:`top: ${ banner.el.offsetHeight+5 }px;`};const el=etc.parseDOM(common.parseHTML`<span ${ attrs }>
				<b class="admin">
					${ msg }
				</b>
			</span>`);banner.el.after(el);this.setElement(el);return this;}});main.reply('notification',msg => new NotificationView(msg));dispatcher[common.NOTIFICATION]=msg => new NotificationView(msg[0]);dispatcher[common.UPDATE_BANNER]=msg => banner.renderInfo(msg[0]);dispatcher[common.RADIO]=msg => options.get('nowPlaying')&&!main.isMobile&&banner.renderRadio(msg[0]);}};});
//# sourceMappingURL=maps/banner.js.map
