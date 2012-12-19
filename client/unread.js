(function () {

var normalTitle = document.title;
var updateCount = 0;
var blurred = false;

window.addEventListener('focus', function () {
	blurred = false;
	updateCount = 0;
	document.title = normalTitle;
}, false);

window.addEventListener('blur', function () {
	blurred = true;
	updateCount = 0;
}, false);

oneeSama.hook('afterInsert', function () {
	if (!blurred)
		return;
	updateCount++;
	document.title = '(' + updateCount + ') ' + normalTitle;
});

})();
