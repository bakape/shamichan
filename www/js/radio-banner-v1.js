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
	
	(function build_faq(){
		var answers = [
			"Lewd is good", 
			"No 3D porn", 
			"Image size limit is 20 MB", 
			"mumble.meguca.org:64738", 
			"github.com/bakape/doushio", 
			"The admin is drunk"
		];
		var list = ['<ul>'];
		answers.forEach(function(entry){
			list.push('<li>' + entry + '<li>');
		});
		list.push('<ul>');
		document.getElementById('FAQ').innerHTML = list.join('');
	})();
};

function FAQ_onclick(){
	if ($('#FAQ').css('display') == 'none')
		$('#FAQ').show();
	else
		$('#FAQ').hide();
}
