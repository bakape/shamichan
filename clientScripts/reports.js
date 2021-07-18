// Use only ES5
(function() {
    // Create an entry for the reports table from server sent data
    function processEvent(sseData) {
        var n = sseData.Post
        var tbl = document.querySelector("tbody");
        var row = document.createElement("tr");
        row.innerHTML = 
            '<td></td>' + 
            '<td>' +
            '<a class="post-link" data-id="' + n + '" href="/all/' + n + '#p' + n + '">>>' + n + '</a>' +
            '<a class="hash-link" href="/all/' + n + '#p' + n + '"> #</a>' +
            '</td>' + 
            '<td>' + sseData.Reason + '</td>' +
            '<td>recently!</td>';
        tbl.insertBefore(row, tbl.firstChild.nextSibling);
    }

	function loadScript(path) {
		var head = document.getElementsByTagName('head')[0];
		var script = document.createElement('script');
		script.type = 'text/javascript';
		script.src = '/assets/' + path + '.js';
		head.appendChild(script);
		return script;
	}

	loadScript("js/static/main").onload = function () {
		require("client/sse/index").default(processEvent);
	};
})();
