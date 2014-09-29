window.onload = function(){
	var info;
	(function write_banner(){
	    $.getJSON('https://r-a-d.io/api', function(data){
	        var main = data.main;
	        var new_info ='<a href="http://r-a-d.io/" target="_blank">' + '[' + main.listeners + '] ' +
	            main.dj.djname + '</a>' + '&nbsp;&nbsp;' + main.np;
			if (new_info != info){
				info = new_info;
	        	document.getElementById('banner_center').innerHTML = info;
	       	}
	       	setTimeout(write_banner, 10000);
	    });
	})();
};
