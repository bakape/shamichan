#include "../lang.hh"
#include "../options/main.hh"
#include "../state.hh"
#include "../util.hh"
#include "models.hh"
#include "view.hh"

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

    return n;
}

// Render a link to the image search provider
static Node image_search_link(int i, const std::string& url)
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
    std::string root;
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
    std::string url;
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
