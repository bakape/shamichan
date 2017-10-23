#include "../../brunhild/util.hh"
#include "../lang.hh"
#include "../options/options.hh"
#include "../state.hh"
#include "../util.hh"
#include "models.hh"
#include "view.hh"
#include <iomanip>
#include <sstream>

using brunhild::escape;
using std::ostringstream;

Node PostView::render_figcaption(const Image& img)
{
    Node n = { "figcaption", { { "class", "spaced" } } };
    n.children.reserve(4);

    if (options->hide_thumbs || options->work_mode_toggle) {
        n.children.push_back({
            "a",
            {
                { "class", "act" },
            },
            lang->posts.at(reveal_thumbnail ? "hide" : "show"),
        });
    }
    if (img.thumb_type != FileType::no_file && img.file_type != FileType::pdf) {
        n.children.push_back(render_image_search(img));
    }
    n.children.push_back(render_file_info(img));

    // File name + download link
    auto& ext = file_extentions.at(img.file_type);
    ostringstream name, url;
    name << escape(img.name) << '.' << ext;
    url << "/assets/images/src/" << img.SHA1 << '.' << ext;
    n.children.push_back({ "a",
        {
            { "href", url.str() }, { "download", name.str() },
        },
        name.str() });

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
    ostringstream unencoded, url;
    unencoded << *location_origin << "/assets/images/" << root << '/'
              << img.SHA1 << '.' << file_extentions.at(typ);
    url << url_encode(unencoded.str());

    const bool enabled[6] = {
        options->google, options->iqdb, options->sauce_nao, options->what_anime,
        options->desu_storage, options->exhentai,
    };
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

Node PostView::render_file_info(const Image& img)
{
    using std::setw;

    ostringstream s;
    bool first = true;
    s << '(';

// Appends a comma and a space after the first invocation
#define comma()                                                                \
    if (!first) {                                                              \
        s << ", ";                                                             \
    } else {                                                                   \
        first = false;                                                         \
    }

    if (img.artist) {
        comma();
        s << escape(*img.artist);
    }
    if (img.title) {
        comma();
        s << escape(*img.title);
    }
    if (img.audio) {
        comma();
        s << "♫";
    }
    if (img.length) {
        comma();
        if (img.length < 60) {
            s << "0:" << setw(2) << img.length;
        } else {
            s << setw(2) << img.length / 60 << ':' << setw(2)
              << img.length % 60;
        }
    }
    if (img.apng) {
        comma();
        s << "APNG";
    }

    // Readable file size
    comma();
    if (img.size < 1 << 10) {
        s << img.size << " B";
    } else if (img.size < 1 << 20) {
        s << img.size / (1 << 10) << " KB";
    } else {
        s << std::setprecision(1) << std::fixed << (float)img.size / (1 << 20)
          << " MB";
    }

    // Media dimensions
    if (const auto[w, h, _, __] = img.dims; w && h) {
        comma();
        s << w << 'x' << h;
    }

    s << ')';
    return Node("span", s.str());
}

Node PostView::render_image(const Image& img)
{
    const std::string src = img.source_path();
    std::string thumb;
    uint16_t h, w;

    if (img.thumb_type == FileType::no_file) {
        // No thumbnail exists. Assign default.
        std::string file;
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

    // Downscale thumbnail for higher DPI, unless specified not to
    if (!large_thumbnail && (w > 125 || h > 125)) {
        w *= 0.8333;
        h *= 0.8333;
    }

    return {
        "figure", {},
        { {
            "a",
            {
                { "href", src }, { "target", "_blank" },
            },
            { {
                "img",
                {
                    { "src", thumb }, { "width", std::to_string(w) },
                    { "height", std::to_string(h) },
                },
            } },
        } },
    };
}
