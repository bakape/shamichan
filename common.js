function escape_html(html) {
	return html.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(
		/>/g, '&gt;').replace(/"/g, '&quot;');
}

function flatten(frags) {
	var out = [];
	for (var i = 0; i < frags.length; i++) {
		var frag = frags[i];
		var t = typeof(frag);
		if (t == 'object' && typeof(frag.safe) == 'string')
			out.push(frag.safe);
		else if (frag.constructor == Array)
			out = out.concat(flatten(frag));
		else if (t == 'string')
			out.push(escape_html(frag));
		else if (t == 'number')
			out.push(frag.toString())
		else
			out.push('???');
	}
	return out;
}
exports.flatten = flatten;

safe = function (frag) {
	return {safe: frag};
}
exports.safe = safe;

function map_unsafe(frags, func) {
	for (var i = 0; i < frags.length; i++) {
		if (typeof(frags[i]) == 'string')
			frags[i] = func(frags[i]);
		else if (frags[i].constructor == Array)
			frags[i] = map_unsafe(frags[i], func);
	}
	return frags;
}

function format_line(line, context) {
	line = line.split(/(\[\/?spoiler\])/i);
	for (var i = 0; i < line.length; i++) {
		if (i % 2 == 0)
			continue;
		if (line[i][1] != '/') {
			context.spoilers++;
			line[i] = safe('<del>');
		}
		else if (context.spoilers > 0) {
			context.spoilers--;
			line[i] = safe('</del>');
		}
	}
	map_unsafe(line, function (frag) {
		var gt = frag.indexOf('>');
		if (gt >= 0)
			return [frag.substr(0, gt), safe('<em>'),
				frag.substr(gt), safe('</em>')];
		return frag;
	});
	return line;
}
exports.format_line = format_line;

function format_body(body) {
	var lines = body.split('\n');
	var output = [];
	var context = {spoilers: 0};
	for (var i = 0; i < lines.length; i++) {
		var line = format_line(lines[i], context);
		if (line.length > 1 || line[0] != '')
			output.push(line);
		output.push(safe('<br>'));
	}
	while (output.length && output[output.length-1].safe == '<br>')
		output.pop();
	for (var i = 0; i < context.spoilers; i++)
		output.push(safe('</del>'));
	return output;
}

function time_to_str(time) {
	function pad_zero(n) { return (n < 10 ? '0' : '') + n; }
	return pad_zero(time[0]) + ':' + pad_zero(time[1]);
}

exports.gen_post_html = function (data) {
	var post = [safe('\t\t<li name="q'), data.num, safe('"><span><b>'),
		data.name, safe('</b> <code>'), (data.trip || ''),
		safe('</code> <time>'), time_to_str(data.time),
		safe('</time> No.'), data.num, safe('</span> <blockquote>'),
		format_body(data.body), safe('</blockquote></li>\n')];
	return flatten(post).join('');
}

exports.parse_name = function (name) {
	var hash = name.indexOf('#');
	var tripcode = null;
	if (hash >= 0) {
		tripcode = name.substr(hash+1);
		name = name.substr(0, hash);
	}
	return [name.trim() || 'Anonymous', tripcode];
}

function clone (obj) {
	if (obj == null || typeof(obj) != 'object')
		return obj;
	var temp = new obj.constructor();
	for (var key in obj)
		temp[key] = clone(obj[key]);
	return temp;
}
exports.clone = clone;
