(function() {
	$('#catalogLink').click(function() {
		fetchJSON(render);
	});

	function fetchJSON(cb) {
		$.getJSON(config.API_URL + 'catalog/' + BOARD, function(data) {
			if (!data)
				alert('Error');
			cb(data);
		});
	}

	function render(json) {
		var $start = $('aside.act').first();
		var $el = $('<div/>', {
			id: 'catalog',
		});
		// Remove threads
		$start.nextUntil($('hr.sectionHr').last()).remove();
		$('.pagination').html('<a onclick="location.reload();">Return</a>');

		var data,$post, html;
		for (var i = 0; i < json.length; i++) {
			data = json[i];
			$post = $('<article/>');
			html = [];
			data.dims = data.dims.split(',');
			// Downscale thumbnail
			data.dims[2] /= 1.66;
			data.dims[3] /= 1.66;
			// Render thumbnail
			html.push(oneeSama.gazou_img(data, false, './' + data.num).html, safe('<br>'));
			html.push(safe('<small>R: ' + data.replies + ' ' + oneeSama.expansion_links_html(data.num) + '<br></small>'));
			if (data.subject)
				html.push(safe('<h3>「' + data.subject + '」</h3>'));
			// Render text body
			html.push(oneeSama.karada(data.body));

			$post.append(flatten(html).join(''));
			// Prevent image hover preview
			$post.find('img').addClass('expanded');
			$el.append($post);
		}
		// Prevent insertion of new threads
		BUMP = false;
		$start.after($el);
	}
})();
