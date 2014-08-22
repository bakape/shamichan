function write_banner(){
    $.getJSON('http://r-a-d.io/api', function(data){
        var main = data.main;
        var info ='<b>' + '<a href="http://r-a-d.io/" target="_blank">' + '[' + main.listeners + '] ' +
        	main.dj.djname + '</a>' + '&nbsp;&nbsp;' + main.np + '</b>';
        document.getElementById('banner').innerHTML = info;
    });
}
write_banner();
setInterval(write_banner, 10000);
