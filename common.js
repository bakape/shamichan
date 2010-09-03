function escape_html(html) {
	return html.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(
		/>/g, '&gt;').replace(/"/g, '&quot;');
}

function time_to_str(time) {
	function pad_zero(n) { return (n < 10 ? '0' : '') + n; }
	return pad_zero(time[0]) + ':' + pad_zero(time[1]);
}

exports.gen_post_html = function (data) {
	var body = escape_html(data.body).replace(/\n/g, '<br>');
	return '\t<li name="post' + data.num + '"><span><b>' +
		escape_html(data.name || 'Anonymous') + '</b> <code>' +
		escape_html(data.trip) + '</code> <time>' +
		time_to_str(data.time) + '</time> No.' + data.num +
		'</span> <blockquote>' + body + '</blockquote></li>\n';
}
