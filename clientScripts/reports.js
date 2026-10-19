// Use only ES5
(function() {
    // Create an entry for the reports table from server sent data
    function processEvent(sseData) {
        var n = sseData.Post
        var tbl = document.querySelector("tbody");
        var row = document.createElement("tr");
        
		let postLink = document.createElement("a");
        postLink.className = "post-link";
        postLink.dataset.id = n;
        postLink.href = '/all/' + n + '#p' + n;
        postLink.textContent = '>>' + n;

        let hashLink = document.createElement("a");
        hashLink.className = "hash-link";
        hashLink.href = '/all/' + n + '#p' + n;
        hashLink.textContent = ' #';

        for (let i = 0; i < 4; i++) {
            let field = document.createElement("td");
            switch (i) {
                case 0:
                    break;
                case 1:
                    field.appendChild(postLink);
                    field.appendChild(hashLink);
                    break;
                case 2:
                    field.textContent = sseData.Reason;
                    break;
                case 3:
                    field.textContent = 'recently!';
                    break;
                }
                row.appendChild(field);
        }
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
