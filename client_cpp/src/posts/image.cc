#include "../../brunhild/events.hh"
#include "../../brunhild/mutations.hh"
#include "../../brunhild/util.hh"
#include "../lang.hh"
#include "../options/options.hh"
#include "../state.hh"
#include "../util.hh"
#include "etc.hh"
#include "view.hh"
#include <emscripten.h>
#include <emscripten/bind.h>
#include <iomanip>
#include <sstream>

using brunhild::escape;
using std::optional;
using std::ostringstream;
using std::string;

// TODO: Expand all images and automatic image expansion

Node PostView::render_figcaption()
{
    auto& img = *m->image;

    Node n = { "figcaption", { { "class", "spaced" } } };
    n.children.reserve(4);

    if (options.hide_thumbs || options.work_mode_toggle) {
        n.children.push_back({
            "a",
            {
                { "class", "image-toggle act" },
                { "data-id", std::to_string(m->id) },
            },
            lang.posts.at(reveal_thumbnail ? "hide" : "show"),
        });
    }
    if (img.thumb_type != FileType::no_file && img.file_type != FileType::pdf) {
        n.children.push_back(render_image_search());
    }
    n.children.push_back(render_file_info());

    // File name + download link
    auto& ext = file_extentions.at(img.file_type);
    ostringstream name, url;
    name << escape(img.name) << '.' << ext;
    url << "/assets/images/src/" << img.SHA1 << '.' << ext;
    n.children.push_back({ "a",
        { { "href", url.str() }, { "download", name.str() } }, name.str() });

    n.stringify_subtree();
    return n;
}

// Render a link to the image search provider
static Node image_search_link(int i, const string& url)
{
    const static char* abbrev[6] = { "G", "Iq", "Sn", "Wa", "Ds", "Ex" };
    const static char* url_starts[6] = {
        "https://www.google.com/searchbyimage?image_url=",
        "http://iqdb.org/?url=",
        "http://saucenao.com/search.php?db=999&url=",
        "https://trace.moe/?url=",
        "https://desuarchive.org/_/search/image/",
        "http://exhentai.org/?fs_similar=1&fs_exp=1&f_shash=",
    };

    return Node("a",
        {
            { "target", "_blank" },
            { "rel", "nofollow" },
            { "href", url_starts[i] + url },
        },
        abbrev[i]);
}

Node PostView::render_image_search()
{
    auto const& img = *m->image;
    Node n = {
        "span",
        {
            { "class", "spaced" },
            // TODO: Rework the CSS for this class
            { "style", "font-weight: 700;" },
        },
    };
    n.children.reserve(6);

    // Resolve URL of image search providers, that require to download the
    // image file
    string root;
    FileType typ;
    switch (img.file_type) {
    case FileType::jpg:
    case FileType::gif:
    case FileType::png:
        if (img.size < 8 << 20) { // Limit on many providers
            root = "src";
            typ = img.file_type;
        }
        break;
    }
    if (root == "") {
        root = "thumb";
        typ = img.thumb_type;
    }
    ostringstream unencoded, url;
    unencoded << location_origin << "/assets/images/" << root << '/' << img.SHA1
              << '.' << file_extentions.at(typ);
    url << url_encode(unencoded.str());

    const bool enabled[6] = { options.google, options.iqdb, options.sauce_nao,
        options.what_anime, options.desu_storage, options.exhentai };
    for (int i = 0; i < 4; i++) {
        if (enabled[i]) {
            n.children.push_back(image_search_link(i, url.str()));
        }
    }
    if (enabled[4]) {
        switch (img.file_type) {
        case FileType::jpg:
        case FileType::png:
        case FileType::gif:
        case FileType::webm:
            n.children.push_back(image_search_link(4, url.str()));
        }
    }
    if (enabled[5]) {
        switch (img.file_type) {
        case FileType::jpg:
        case FileType::png:
            n.children.push_back(image_search_link(5, url.str()));
        }
    }

    return n;
}

Node PostView::render_file_info()
{
    using std::setw;

    auto& img = *m->image;
    ostringstream s;
    bool first = true;
    s << '(';

// Appends a comma and a space after the first invocation
#define COMMA                                                                  \
    if (!first) {                                                              \
        s << ", ";                                                             \
    } else {                                                                   \
        first = false;                                                         \
    }

    if (img.artist) {
        COMMA
        s << escape(*img.artist);
    }
    if (img.title) {
        COMMA
        s << escape(*img.title);
    }
    if (img.audio) {
        COMMA
        s << "♫";
    }
    if (img.length) {
        COMMA
        if (img.length < 60) {
            s << "0:" << setw(2) << img.length;
        } else {
            s << setw(2) << img.length / 60 << ':' << setw(2)
              << img.length % 60;
        }
    }
    if (img.apng) {
        COMMA
        s << "APNG";
    }

    // Readable file size
    COMMA
    if (img.size < 1 << 10) {
        s << img.size << " B";
    } else if (img.size < 1 << 20) {
        s << img.size / (1 << 10) << " KB";
    } else {
        s << std::setprecision(1) << std::fixed << (float)img.size / (1 << 20)
          << " MB";
    }

    // Media dimensions
    if (const auto [w, h, _, __] = img.dims; w && h) {
        COMMA
        s << w << 'x' << h;
    }

    s << ')';
    return Node("span", s.str());
}

// Render unexpanded file thumbnail image
static Node render_thumbnail(const Image& img)
{
    string thumb;
    uint16_t h, w;

    if (img.thumb_type == FileType::no_file) {
        // No thumbnail exists. Assign default.
        string file;
        switch (img.file_type) {
        case FileType::mp4:
        case FileType::mp3:
        case FileType::ogg:
        case FileType::flac:
            file = "audio";
            break;
        default:
            file = "file";
        }
        thumb = "/assets/" + file + ".png";
        h = w = 150;
    } else if (img.spoiler) {
        thumb = "/assets/spoil/default.jpg";
        h = w = 150;
    } else {
        thumb = img.thumb_path();
        w = img.dims[2];
        h = img.dims[3];
    }

    return {
        "img",
        {
            { "src", thumb },
            { "width", std::to_string(w) },
            { "height", std::to_string(h) },
        },
    };
}

// Format audio volume option setter to string
static string format_volume_setter()
{
    return "this.volume=" + std::to_string((float)options.audio_volume / 100);
}

// Render expanded file image, video or audio
static void render_expanded(
    const Image& img, Node& inner, optional<Node>& audio)
{
    const auto src = img.source_path();

    switch (img.file_type) {
    case FileType::ogg:
    case FileType::mp4:
        // Can have only audio
        if (img.video) {
            goto render_video;
        }
    case FileType::flac:
    case FileType::mp3:
        // Audio controls are rendered outside the figure. Keep the
        // thumbnail.
        audio = {
            {
                "audio",
                {
                    { "autoplay", "" },
                    { "controls", "" },
                    { "loop`", "" },
                    { "src", src },
                    { "onloadstart", format_volume_setter() },
                },
            },
        };
        inner = render_thumbnail(img);
        return;
    case FileType::webm:
    render_video:
        inner = {
            "video",
            {
                { "autoplay", "" },
                { "controls", "" },
                { "loop`", "" },
                { "onloadstart", format_volume_setter() },
            },
        };
        break;
    default:
        inner = { "img" };
    }

    inner.attrs["class"] = options.inline_fit == Options::FittingMode::width
        ? "fit-to-width"
        : "fit-to-screen";
    inner.attrs["src"] = src;
}

std::tuple<Node, optional<Node>> PostView::render_image()
{
    auto& img = *m->image;
    Node inner;
    optional<Node> audio;

    if (expanded) {
        render_expanded(img, inner, audio);
    } else {
        inner = render_thumbnail(img);
    }

    const string id_str = std::to_string(m->id);
    inner.attrs["data-id"] = id_str;
    Node n({
        "figure",
        {},
        {
            {
                "a",
                {
                    { "href", img.source_path() },
                    { "target", "_blank" },
                    { "data-id", id_str },
                },
                { inner },
            },
        },
    });
    n.stringify_subtree();
    return { n, audio };
}

// Match view with image or return
#define MATCH_WITH_IMAGE(event)                                                \
    auto res = match_view(event);                                              \
    if (!res) {                                                                \
        return;                                                                \
    }                                                                          \
    auto [model, view] = *res;                                                 \
    if (!model->image) {                                                       \
        return;                                                                \
    }

void handle_image_click(emscripten::val& event)
{
    if (page.catalog) {
        return;
    }
    MATCH_WITH_IMAGE(event);
    auto& img = *model->image;

    // Simply download the file
    switch (img.file_type) {
    case FileType::pdf:
    case FileType::zip:
    case FileType::_7z:
    case FileType::targz:
    case FileType::tarxz:
    case FileType::txt:
        EM_ASM_INT(
            {
                if (!document.querySelector) {
                    // Really old browser. Fuck it!
                    return;
                }
                document.getElementById(UTF8ToString($0))
                    .querySelector('figcaption a[download]')
                    .click();
            },
            view->id.data());
        return;
    }

    view->expanded = !view->expanded;
    if (options.inline_fit == Options::FittingMode::width
        && !options.gallery_mode_toggle
        && img.dims[1]
            > emscripten::val::global("window")["innerHeight"].as<unsigned>()) {
        brunhild::scroll_into_view(view->id);
    }
    view->patch();
}

void toggle_hidden_thumbnail(emscripten::val& event)
{
    MATCH_WITH_IMAGE(event);
    view->reveal_thumbnail = !view->reveal_thumbnail;
    view->patch();
}
