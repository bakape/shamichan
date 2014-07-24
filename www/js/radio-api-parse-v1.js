function write_banner(){
    $.getJSON('http://r-a-d.io/api', function(data){
        var main = data.main;
        var info = '<a href="http://r-a-d.io/" target="_blank">' + "DJ: " + main.djname + " L: " + main.listeners + '</a>' + '&nbsp;&nbsp;&nbsp;&nbsp;' + main.np;
        document.getElementById('radioBanner').innerHTML = info;
    });
}

function rb_init(){
    write_banner();
    setInterval(write_banner, 10000);
}
