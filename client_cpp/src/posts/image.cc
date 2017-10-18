#include "../lang.hh"
#include "../options/main.hh"
#include "../state.hh"
#include "../util.hh"
#include "models.hh"
#include "view.hh"
#include <sstream>

using brunhild::escape;

Node PostView::render_figcaption(const Image& img, bool reveal)
{
    Node n = { "figcaption", { { "class", "spaced" } } };
    n.children.reserve(4);

    if (options->hide_thumbs || options->work_mode_toggle) {
        n.children.push_back({
            "a",
            {
                { "class", "act" },
            },
            lang->posts.at(reveal ? "hide" : "show"),
        });
    }
    if (img.thumb_type != FileType::no_file && img.file_type != FileType::pdf) {
        n.children.push_back(render_image_search(img));
    }
    n.children.push_back(render_file_info(img));

    // File name + download link
    auto& ext = file_extentions.at(img.file_type);
    string name, url;
    name.reserve(img.name.size() * 1.2);
    escape(name, img.name);
    name += '.';
    name += ext;
    url.reserve(72);
    url += "/assets/images/src/";
    url += img.SHA1;
    url += '.';
    url += ext;
    n.children.push_back({ "a",
        {
            { "href", url }, { "download", name },
        },
        name });

    return n;
}

// Render a link to the image search provider
static Node image_search_link(int i, const string& url)
{
    const static string abbrev[6] = { "G", "Iq", "Sn", "Wa", "Ds", "Ex" };
    const static string url_starts[6] = {
        "https://www.google.com/searchbyimage?image_url=",
        "http://iqdb.org/?url=", "http://saucenao.com/search.php?db=999&url=",
        "https://whatanime.ga/?url=", "https://desuarchive.org/_/search/image/",
        "http://exhentai.org/?fs_similar=1&fs_exp=1&f_shash=",
    };

    return Node("a",
        {
            { "target", "_blank" }, { "rel", "nofollow" },
            { "href", url_starts[i] + url },
        },
        abbrev[i]);
}

Node PostView::render_image_search(const Image& img)
{
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
    string url;
    url.reserve(128);
    url += *location_origin;
    url += "/assets/images/";
    url += root;
    url += '/';
    url += img.SHA1;
    url += '.';
    url += file_extentions.at(typ);
    url = url_encode(url);

    const bool enabled[6] = {
        options->google, options->iqdb, options->sauce_nao, options->what_anime,
        options->desu_storage, options->exhentai,
    };
    for (int i = 0; i < 4; i++) {
        if (enabled[i]) {
            n.children.push_back(image_search_link(i, url));
        }
    }
    if (enabled[4]) {
        switch (img.file_type) {
        case FileType::jpg:
        case FileType::png:
        case FileType::gif:
        case FileType::webm:
            n.children.push_back(image_search_link(4, url));
        }
    }
    if (enabled[5]) {
        switch (img.file_type) {
        case FileType::jpg:
        case FileType::png:
            n.children.push_back(image_search_link(5, url));
        }
    }

    return n;
}

Node PostView::render_file_info(const Image& img)
{
    using std::to_string;

    string s;
    s.reserve(32);
    bool first = true;
    s += '(';

// Appends a comma and a space after the first invocation
#define comma()                                                                \
    if (!first) {                                                              \
        s += ", ";                                                             \
    } else {                                                                   \
        first = false;                                                         \
    }

    if (img.artist) {
        comma();
        escape(s, *img.artist);
    }
    if (img.title) {
        comma();
        escape(s, *img.title);
    }
    if (img.audio) {
        comma();
        s += "♫";
    }
    if (img.length) {
        comma();
        if (img.length < 60) {
            s += "0:";
            pad(s, img.length);
        } else {
            pad(s, img.length / 60);
            s += ':';
            pad(s, img.length % 60);
        }
    }
    if (img.apng) {
        comma();
        s += "APNG";
    }

    // Readable file size
    comma();
    if (img.size < 1 << 10) {
        s += to_string(img.size);
        s += " B";
    } else if (img.size < 1 << 20) {
        s += to_string(img.size / (1 << 10));
        s += " KB";
    } else {
        std::ostringstream str;
        str << std::setprecision(1) << std::fixed
            << (float)img.size / (1 << 20);
        s += str.str();
        s += " MB";
    }

    // Media dimensions
    if (const auto[w, h, _, __] = img.dims; w && h) {
        comma();
        s += to_string(w);
        s += 'x';
        s += to_string(h);
    }

    s += ')';
    return Node("span", s);
}
