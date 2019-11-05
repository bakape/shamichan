// Code generated by qtc from "board.html". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line board.html:1
package templates

//line board.html:1
import "strconv"

//line board.html:2
import "fmt"

//line board.html:3
import "github.com/Chiiruno/meguca/config"

//line board.html:4
import "github.com/Chiiruno/meguca/common"

//line board.html:5
import "github.com/Chiiruno/meguca/lang"

//line board.html:6
import "github.com/Chiiruno/meguca/imager/assets"

//line board.html:7
import ass "github.com/Chiiruno/meguca/assets"

//line board.html:9
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line board.html:9
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line board.html:9
func streamrenderBoard(qw422016 *qt422016.Writer, threadHTML []byte, id, title string, conf config.BoardConfContainer, page, total int, pos common.ModerationLevel, catalog bool) {
//line board.html:10
	ln := lang.Get()

//line board.html:11
	bannerID, mime, ok := ass.Banners.Random(conf.ID)

//line board.html:12
	if ok {
//line board.html:12
		qw422016.N().S(`<h1 class="image-banner">`)
//line board.html:14
		streamasset(qw422016, fmt.Sprintf("/assets/banners/%s/%d", conf.ID, bannerID), mime)
//line board.html:14
		qw422016.N().S(`</h1>`)
//line board.html:16
	}
//line board.html:16
	qw422016.N().S(`<h1 id="page-title">`)
//line board.html:18
	qw422016.N().S(title)
//line board.html:18
	qw422016.N().S(`</h1><span class="aside-container"><aside id="thread-form-container" class="glass"><span class="act"><a class="new-thread-button">`)
//line board.html:24
	qw422016.N().S(ln.Common.UI["newThread"])
//line board.html:24
	qw422016.N().S(`</a></span><form id="new-thread-form" action="/api/create-thread" method="post" enctype="multipart/form-data" class="hidden">`)
//line board.html:28
	if id == "all" {
//line board.html:28
		qw422016.N().S(`<select name="board" required>`)
//line board.html:30
		for _, b := range config.GetBoardTitles() {
//line board.html:31
			if b.ID == "all" {
//line board.html:32
				continue
//line board.html:33
			}
//line board.html:33
			qw422016.N().S(`<option value="`)
//line board.html:34
			qw422016.N().S(b.ID)
//line board.html:34
			qw422016.N().S(`">`)
//line board.html:35
			streamformatTitle(qw422016, b.ID, b.Title)
//line board.html:35
			qw422016.N().S(`</option>`)
//line board.html:37
		}
//line board.html:37
		qw422016.N().S(`</select><br>`)
//line board.html:40
	} else {
//line board.html:40
		qw422016.N().S(`<input type="text" name="board" value="`)
//line board.html:41
		qw422016.N().S(conf.ID)
//line board.html:41
		qw422016.N().S(`" hidden>`)
//line board.html:42
	}
//line board.html:42
	qw422016.N().S(`<input name="subject" placeholder="`)
//line board.html:43
	qw422016.N().S(ln.UI["subject"])
//line board.html:43
	qw422016.N().S(`" required type="text" maxlength="100"><br>`)
//line board.html:45
	streamnoscriptPostCreationFields(qw422016, pos)
//line board.html:46
	if id == "all" || !conf.TextOnly {
//line board.html:47
		streamuploadForm(qw422016)
//line board.html:48
	}
//line board.html:49
	streamcaptcha(qw422016, id)
//line board.html:50
	streamsubmit(qw422016, false)
//line board.html:50
	qw422016.N().S(`</form></aside><aside id="refresh" class="act glass noscript-hide"><a>`)
//line board.html:55
	qw422016.N().S(ln.Common.UI["refresh"])
//line board.html:55
	qw422016.N().S(`</a></aside>`)
//line board.html:58
	streamcatalogLink(qw422016, catalog)
//line board.html:59
	if !catalog {
//line board.html:60
		streampagination(qw422016, page, total)
//line board.html:61
	}
//line board.html:62
	streamhoverReveal(qw422016, "aside", conf.Notice, ln.Common.UI["showNotice"])
//line board.html:63
	streamhoverReveal(qw422016, "aside", conf.Rules, ln.Common.UI["rules"])
//line board.html:63
	qw422016.N().S(`<span id="catalog-controls" class="margin-spaced noscript-hide"><input type="text" name="search" placeholder="`)
//line board.html:65
	qw422016.N().S(ln.Common.UI["search"])
//line board.html:65
	qw422016.N().S(`" title="`)
//line board.html:65
	qw422016.N().S(ln.UI["searchTooltip"])
//line board.html:65
	qw422016.N().S(`">`)
//line board.html:66
	if catalog {
//line board.html:66
		qw422016.N().S(`<select name="sortMode">`)
//line board.html:68
		for i, s := range [...]string{"bump", "lastReply", "creation", "replyCount", "fileCount"} {
//line board.html:68
			qw422016.N().S(`<option value="`)
//line board.html:69
			qw422016.N().S(s)
//line board.html:69
			qw422016.N().S(`">`)
//line board.html:70
			qw422016.N().S(ln.SortModes[i])
//line board.html:70
			qw422016.N().S(`</option>`)
//line board.html:72
		}
//line board.html:72
		qw422016.N().S(`</select>`)
//line board.html:74
	}
//line board.html:74
	qw422016.N().S(`</span></span><hr>`)
//line board.html:78
	qw422016.N().Z(threadHTML)
//line board.html:78
	qw422016.N().S(`<script id="board-configs" type="application/json">`)
//line board.html:80
	qw422016.N().Z(conf.JSON)
//line board.html:80
	qw422016.N().S(`</script><hr><span class="aside-container">`)
//line board.html:84
	streamcatalogLink(qw422016, catalog)
//line board.html:85
	if !catalog {
//line board.html:86
		streampagination(qw422016, page, total)
//line board.html:87
	}
//line board.html:87
	qw422016.N().S(`</span>`)
//line board.html:89
	streamloadingImage(qw422016, conf.ID)
//line board.html:90
}

//line board.html:90
func writerenderBoard(qq422016 qtio422016.Writer, threadHTML []byte, id, title string, conf config.BoardConfContainer, page, total int, pos common.ModerationLevel, catalog bool) {
//line board.html:90
	qw422016 := qt422016.AcquireWriter(qq422016)
//line board.html:90
	streamrenderBoard(qw422016, threadHTML, id, title, conf, page, total, pos, catalog)
//line board.html:90
	qt422016.ReleaseWriter(qw422016)
//line board.html:90
}

//line board.html:90
func renderBoard(threadHTML []byte, id, title string, conf config.BoardConfContainer, page, total int, pos common.ModerationLevel, catalog bool) string {
//line board.html:90
	qb422016 := qt422016.AcquireByteBuffer()
//line board.html:90
	writerenderBoard(qb422016, threadHTML, id, title, conf, page, total, pos, catalog)
//line board.html:90
	qs422016 := string(qb422016.B)
//line board.html:90
	qt422016.ReleaseByteBuffer(qb422016)
//line board.html:90
	return qs422016
//line board.html:90
}

// CatalogThreads renders thread content for a catalog page. Separate function to
// allow caching of generated posts.

//line board.html:94
func StreamCatalogThreads(qw422016 *qt422016.Writer, b []common.Thread, json []byte) {
//line board.html:94
	qw422016.N().S(`<div id="catalog">`)
//line board.html:96
	for _, t := range b {
//line board.html:97
		boardConfig := config.GetBoardConfigs(t.Board)

//line board.html:98
		idStr := strconv.FormatUint(t.ID, 10)

//line board.html:99
		hasImage := t.Image != nil && t.Image.ThumbType != common.NoFile

//line board.html:99
		qw422016.N().S(`<article id="p`)
//line board.html:100
		qw422016.N().S(idStr)
//line board.html:100
		qw422016.N().S(`"`)
//line board.html:100
		qw422016.N().S(` `)
//line board.html:100
		streampostClass(qw422016, t.Post, t.ID)
//line board.html:100
		qw422016.N().S(` `)
//line board.html:100
		qw422016.N().S(`data-id="`)
//line board.html:100
		qw422016.N().S(idStr)
//line board.html:100
		qw422016.N().S(`">`)
//line board.html:101
		streamdeletedToggle(qw422016)
//line board.html:102
		if hasImage {
//line board.html:102
			qw422016.N().S(`<figure>`)
//line board.html:104
			img := *t.Image

//line board.html:104
			qw422016.N().S(`<a href="/`)
//line board.html:105
			qw422016.N().S(t.Board)
//line board.html:105
			qw422016.N().S(`/`)
//line board.html:105
			qw422016.N().S(idStr)
//line board.html:105
			qw422016.N().S(`">`)
//line board.html:106
			if img.Spoiler {
//line board.html:106
				qw422016.N().S(`<img src="/assets/spoil/default.jpg" width="150" height="150" class="catalog">`)
//line board.html:108
			} else {
//line board.html:108
				qw422016.N().S(`<img width="`)
//line board.html:109
				qw422016.N().S(strconv.FormatUint(uint64(img.Dims[2]), 10))
//line board.html:109
				qw422016.N().S(`" height="`)
//line board.html:109
				qw422016.N().S(strconv.FormatUint(uint64(img.Dims[3]), 10))
//line board.html:109
				qw422016.N().S(`" class="catalog" src="`)
//line board.html:109
				qw422016.N().S(assets.ThumbPath(img.ThumbType, img.SHA1))
//line board.html:109
				qw422016.N().S(`">`)
//line board.html:110
			}
//line board.html:110
			qw422016.N().S(`</a></figure>`)
//line board.html:113
		}
//line board.html:113
		qw422016.N().S(`<span class="spaced thread-links hide-empty"><b class="board">/`)
//line board.html:116
		qw422016.N().S(t.Board)
//line board.html:116
		qw422016.N().S(`/</b><span class="counters">`)
//line board.html:119
		qw422016.N().S(strconv.FormatUint(uint64(t.PostCount), 10))
//line board.html:119
		qw422016.N().S(`/`)
//line board.html:121
		qw422016.N().S(strconv.FormatUint(uint64(t.ImageCount), 10))
//line board.html:121
		qw422016.N().S(`</span>`)
//line board.html:123
		if !hasImage {
//line board.html:124
			streamexpandLink(qw422016, t.Board, idStr)
//line board.html:125
		}
//line board.html:126
		streamlast100Link(qw422016, t.Board, idStr)
//line board.html:127
		streamthreadWatcherToggle(qw422016, t.ID)
//line board.html:127
		qw422016.N().S(`</span><br><h3>「`)
//line board.html:131
		qw422016.E().S(t.Subject)
//line board.html:131
		qw422016.N().S(`」</h3><blockquote>`)
//line board.html:134
		streambody(qw422016, t.Post, t.ID, t.Board, false, boardConfig.RbText, boardConfig.Pyu)
//line board.html:134
		qw422016.N().S(`</blockquote></article>`)
//line board.html:137
	}
//line board.html:137
	qw422016.N().S(`<script id="post-data" type="application/json">`)
//line board.html:139
	qw422016.N().Z(json)
//line board.html:139
	qw422016.N().S(`</script></div>`)
//line board.html:142
}

//line board.html:142
func WriteCatalogThreads(qq422016 qtio422016.Writer, b []common.Thread, json []byte) {
//line board.html:142
	qw422016 := qt422016.AcquireWriter(qq422016)
//line board.html:142
	StreamCatalogThreads(qw422016, b, json)
//line board.html:142
	qt422016.ReleaseWriter(qw422016)
//line board.html:142
}

//line board.html:142
func CatalogThreads(b []common.Thread, json []byte) string {
//line board.html:142
	qb422016 := qt422016.AcquireByteBuffer()
//line board.html:142
	WriteCatalogThreads(qb422016, b, json)
//line board.html:142
	qs422016 := string(qb422016.B)
//line board.html:142
	qt422016.ReleaseByteBuffer(qb422016)
//line board.html:142
	return qs422016
//line board.html:142
}

// IndexThreads renders abbreviated threads for display on board index pages

//line board.html:145
func StreamIndexThreads(qw422016 *qt422016.Writer, threads []common.Thread, json []byte) {
//line board.html:146
	root := config.Get().RootURL

//line board.html:147
	bls := extractBacklinks(15*6, threads...)

//line board.html:147
	qw422016.N().S(`<div id="index-thread-container">`)
//line board.html:149
	for _, t := range threads {
//line board.html:150
		idStr := strconv.FormatUint(t.ID, 10)

//line board.html:150
		qw422016.N().S(`<section class="index-thread`)
//line board.html:151
		if t.IsDeleted() {
//line board.html:151
			qw422016.N().S(` `)
//line board.html:151
			qw422016.N().S(`deleted`)
//line board.html:151
		}
//line board.html:151
		qw422016.N().S(`" data-id="`)
//line board.html:151
		qw422016.N().S(idStr)
//line board.html:151
		qw422016.N().S(`">`)
//line board.html:152
		streamdeletedToggle(qw422016)
//line board.html:153
		streamrenderThreadPosts(qw422016, t, bls, root, true)
//line board.html:153
		qw422016.N().S(`<hr></section>`)
//line board.html:156
	}
//line board.html:156
	qw422016.N().S(`<script id="post-data" type="application/json">`)
//line board.html:158
	qw422016.N().Z(json)
//line board.html:158
	qw422016.N().S(`</script>`)
//line board.html:160
	streamencodeBacklinks(qw422016, bls)
//line board.html:160
	qw422016.N().S(`</div>`)
//line board.html:162
}

//line board.html:162
func WriteIndexThreads(qq422016 qtio422016.Writer, threads []common.Thread, json []byte) {
//line board.html:162
	qw422016 := qt422016.AcquireWriter(qq422016)
//line board.html:162
	StreamIndexThreads(qw422016, threads, json)
//line board.html:162
	qt422016.ReleaseWriter(qw422016)
//line board.html:162
}

//line board.html:162
func IndexThreads(threads []common.Thread, json []byte) string {
//line board.html:162
	qb422016 := qt422016.AcquireByteBuffer()
//line board.html:162
	WriteIndexThreads(qb422016, threads, json)
//line board.html:162
	qs422016 := string(qb422016.B)
//line board.html:162
	qt422016.ReleaseByteBuffer(qb422016)
//line board.html:162
	return qs422016
//line board.html:162
}

// Render noscript-specific post creation fields

//line board.html:165
func streamnoscriptPostCreationFields(qw422016 *qt422016.Writer, pos common.ModerationLevel) {
//line board.html:166
	ln := lang.Get()

//line board.html:167
	if pos > common.NotStaff {
//line board.html:168
		streaminput(qw422016, staffTitleSpec.wrap(), ln)
//line board.html:169
	}
//line board.html:170
	for _, s := range specs["noscriptPostCreation"] {
//line board.html:171
		streaminput(qw422016, s, ln)
//line board.html:172
	}
//line board.html:173
}

//line board.html:173
func writenoscriptPostCreationFields(qq422016 qtio422016.Writer, pos common.ModerationLevel) {
//line board.html:173
	qw422016 := qt422016.AcquireWriter(qq422016)
//line board.html:173
	streamnoscriptPostCreationFields(qw422016, pos)
//line board.html:173
	qt422016.ReleaseWriter(qw422016)
//line board.html:173
}

//line board.html:173
func noscriptPostCreationFields(pos common.ModerationLevel) string {
//line board.html:173
	qb422016 := qt422016.AcquireByteBuffer()
//line board.html:173
	writenoscriptPostCreationFields(qb422016, pos)
//line board.html:173
	qs422016 := string(qb422016.B)
//line board.html:173
	qt422016.ReleaseByteBuffer(qb422016)
//line board.html:173
	return qs422016
//line board.html:173
}

// Render image upload form

//line board.html:176
func streamuploadForm(qw422016 *qt422016.Writer) {
//line board.html:176
	qw422016.N().S(`<span class="upload-container"><span data-id="spoiler"><label><input type="checkbox" name="spoiler">`)
//line board.html:181
	qw422016.N().S(lang.Get().Common.Posts["spoiler"])
//line board.html:181
	qw422016.N().S(`</label></span><br><input type="file" name="image" accept="image/png, image/gif, image/jpeg, video/webm, video/ogg, audio/ogg, application/ogg, video/mp4, audio/mp4, audio/mp3, application/zip, application/x-7z-compressed, application/x-xz, application/x-gzip, audio/x-flac, text/plain, application/pdf, video/quicktime, audio/x-flac"><br></span>`)
//line board.html:188
}

//line board.html:188
func writeuploadForm(qq422016 qtio422016.Writer) {
//line board.html:188
	qw422016 := qt422016.AcquireWriter(qq422016)
//line board.html:188
	streamuploadForm(qw422016)
//line board.html:188
	qt422016.ReleaseWriter(qw422016)
//line board.html:188
}

//line board.html:188
func uploadForm() string {
//line board.html:188
	qb422016 := qt422016.AcquireByteBuffer()
//line board.html:188
	writeuploadForm(qb422016)
//line board.html:188
	qs422016 := string(qb422016.B)
//line board.html:188
	qt422016.ReleaseByteBuffer(qb422016)
//line board.html:188
	return qs422016
//line board.html:188
}

// Link to catalog or board page

//line board.html:191
func streamcatalogLink(qw422016 *qt422016.Writer, catalog bool) {
//line board.html:192
	ln := lang.Get().Common.UI

//line board.html:192
	qw422016.N().S(`<aside class="act glass">`)
//line board.html:194
	if catalog {
//line board.html:194
		qw422016.N().S(`<a href=".">`)
//line board.html:196
		qw422016.N().S(ln["return"])
//line board.html:196
		qw422016.N().S(`</a>`)
//line board.html:198
	} else {
//line board.html:198
		qw422016.N().S(`<a href="catalog">`)
//line board.html:200
		qw422016.N().S(ln["catalog"])
//line board.html:200
		qw422016.N().S(`</a>`)
//line board.html:202
	}
//line board.html:202
	qw422016.N().S(`</aside>`)
//line board.html:204
}

//line board.html:204
func writecatalogLink(qq422016 qtio422016.Writer, catalog bool) {
//line board.html:204
	qw422016 := qt422016.AcquireWriter(qq422016)
//line board.html:204
	streamcatalogLink(qw422016, catalog)
//line board.html:204
	qt422016.ReleaseWriter(qw422016)
//line board.html:204
}

//line board.html:204
func catalogLink(catalog bool) string {
//line board.html:204
	qb422016 := qt422016.AcquireByteBuffer()
//line board.html:204
	writecatalogLink(qb422016, catalog)
//line board.html:204
	qs422016 := string(qb422016.B)
//line board.html:204
	qt422016.ReleaseByteBuffer(qb422016)
//line board.html:204
	return qs422016
//line board.html:204
}

// Links to different pages of the board index

//line board.html:207
func streampagination(qw422016 *qt422016.Writer, page, total int) {
//line board.html:207
	qw422016.N().S(`<aside class="glass spaced">`)
//line board.html:209
	if page != 0 {
//line board.html:210
		if page-1 != 0 {
//line board.html:211
			streampageLink(qw422016, 0, "<<")
//line board.html:212
		}
//line board.html:213
		streampageLink(qw422016, page-1, "<")
//line board.html:214
	}
//line board.html:215
	for i := 0; i < total; i++ {
//line board.html:216
		if i != page {
//line board.html:217
			streampageLink(qw422016, i, strconv.Itoa(i))
//line board.html:218
		} else {
//line board.html:218
			qw422016.N().S(`<b>`)
//line board.html:220
			qw422016.N().D(i)
//line board.html:220
			qw422016.N().S(`</b>`)
//line board.html:222
		}
//line board.html:223
	}
//line board.html:224
	if page != total-1 {
//line board.html:225
		streampageLink(qw422016, page+1, ">")
//line board.html:226
		if page+1 != total-1 {
//line board.html:227
			streampageLink(qw422016, total-1, ">>")
//line board.html:228
		}
//line board.html:229
	}
//line board.html:229
	qw422016.N().S(`</aside>`)
//line board.html:231
}

//line board.html:231
func writepagination(qq422016 qtio422016.Writer, page, total int) {
//line board.html:231
	qw422016 := qt422016.AcquireWriter(qq422016)
//line board.html:231
	streampagination(qw422016, page, total)
//line board.html:231
	qt422016.ReleaseWriter(qw422016)
//line board.html:231
}

//line board.html:231
func pagination(page, total int) string {
//line board.html:231
	qb422016 := qt422016.AcquireByteBuffer()
//line board.html:231
	writepagination(qb422016, page, total)
//line board.html:231
	qs422016 := string(qb422016.B)
//line board.html:231
	qt422016.ReleaseByteBuffer(qb422016)
//line board.html:231
	return qs422016
//line board.html:231
}

// Link to a different paginated board page

//line board.html:234
func streampageLink(qw422016 *qt422016.Writer, i int, text string) {
//line board.html:234
	qw422016.N().S(`<a href="?page=`)
//line board.html:235
	qw422016.N().D(i)
//line board.html:235
	qw422016.N().S(`">`)
//line board.html:236
	qw422016.N().S(text)
//line board.html:236
	qw422016.N().S(`</a>`)
//line board.html:238
}

//line board.html:238
func writepageLink(qq422016 qtio422016.Writer, i int, text string) {
//line board.html:238
	qw422016 := qt422016.AcquireWriter(qq422016)
//line board.html:238
	streampageLink(qw422016, i, text)
//line board.html:238
	qt422016.ReleaseWriter(qw422016)
//line board.html:238
}

//line board.html:238
func pageLink(i int, text string) string {
//line board.html:238
	qb422016 := qt422016.AcquireByteBuffer()
//line board.html:238
	writepageLink(qb422016, i, text)
//line board.html:238
	qs422016 := string(qb422016.B)
//line board.html:238
	qt422016.ReleaseByteBuffer(qb422016)
//line board.html:238
	return qs422016
//line board.html:238
}
