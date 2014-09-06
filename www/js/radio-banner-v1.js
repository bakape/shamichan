function write_banner(cb){
    $.getJSON('http://r-a-d.io/api', function(data){
        var main = data.main;
        var info ='<a href="http://r-a-d.io/" target="_blank">' + '[' + main.listeners + '] ' +
            main.dj.djname + '</a>' + '&nbsp;&nbsp;' + main.np;
        document.getElementById('banner_center').innerHTML = info;
	if (cb)
            cb();
    });
}

function banner_left(){
    var xhr = new XMLHttpRequest();
    xhr.open("GET", "http://meguca.org/static/js/banner_left.html", true);
    xhr.onload = function() {
        var text = xhr.responseText;
        document.getElementById('banner_left').innerHTML = '&nbsp;&nbsp;' + text;
    };
    xhr.send(null);
}

write_banner(banner_left);
setInterval(write_banner, 10000);
setInterval(banner_left, 60000);
