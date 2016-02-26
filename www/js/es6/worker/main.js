'use strict';System.register(['./state','./clients'],function(_export,_context){var fetchConfig,clients;return {setters:[function(_state){fetchConfig=_state.fetchConfig;},function(_clients){clients=_clients;}],execute:function(){self.onfetch=event => event.respondWith(fetch(event.request));}};});
//# sourceMappingURL=../maps/worker/main.js.map
