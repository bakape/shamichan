!function(){var e;try{e=JSON.parse(localStorage.options)}catch(a){}var t=config.hard.HTTP.media,n=e&&e.theme?e.theme:config.defaultCSS;document.getElementById("theme").href=t+"css/"+n+".css?v="+clientHash,window.lang=e&&e.lang||config.lang["default"]}();
//# sourceMappingURL=setup.js.map
