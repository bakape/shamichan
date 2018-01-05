#include "url.hh"
#include "etc.hh"
#include <algorithm>
#include <emscripten.h>
#include <sstream>
#include <tuple>
#include <unordered_map>

using std::nullopt;
using std::optional;
using std::string;
using std::string_view;

// TODO: Embed click handling, fetching and expansion

// Types of supported embed providers
enum class Provider { Youtube, Soundcloud, Vimeo };

// Call into JS to validate URL. Keeps us from depending on a parsing library
// and increasing binary size.
static bool validate_url(string_view url)
{
    return (bool)EM_ASM_INT(
        {
            try {
                new URL(UTF8ToString($0));
                return true;
            } catch (e) {
                return false;
            }
        },
        string(url).c_str());
}

// Formatter for the noembed.com meta-provider
static Node format_noembed(const Provider prov, const string href)
{
    // Names of providers
    char const* name = NULL;
    switch (prov) {
    case Provider::Youtube:
        name = "Youtube";
        break;
    case Provider::Soundcloud:
        name = "SoundCloud";
        break;
    case Provider::Vimeo:
        name = "Vimeo";
        break;
    }
    std::ostringstream s;
    s << '[' << name << "] ???";

    return {
        "em", {},
        {
            {
                "a",
                {
                    { "rel", "noreferrer" }, { "href", brunhild::escape(href) },
                    { "class", "embed" }, { "target", "_blank" },
                },
                s.str(),
            },
        },
    };
}

// Parse for presence of embed links. Returns Node, if valid URL found.
static optional<Node> parse_embeds(string_view word)
{
    if (!validate_url(word)) {
        return nullopt;
    }

    // URL matching patterns for their respective providers
    const static std::tuple<Provider, char const*> patterns[4] = {
        { Provider::Youtube,
            R"'(https?:\/\/(?:[^\.]+\.)?youtube\.com\/watch\/?\?(?:.+&)?v=[^&]+)'" },
        { Provider::Youtube,
            R"'(https?:\/\/(?:[^\.]+\.)?(?:youtu\.be|youtube\.com\/embed)\/[a-zA-Z0-9_-]+)'" },
        { Provider::Soundcloud, R"'(https?:\/\/soundcloud.com\/.*)'" },
        { Provider::Vimeo, R"'(https?:\/\/(?:www\.)?vimeo\.com\/.+)'" },
    };

    const string href(word);
    for (auto && [ prov, pat ] : patterns) {
        // Call into JS, to avoid including <regex> and bloating the binary
        const bool match = (bool)EM_ASM_INT(
            {
                // TODO: Don't recompile RegExp on each call
                return new RegExp(UTF8ToString($0)).test(UTF8ToString($1));
            },
            pat, href.c_str());
        if (match) {
            return { format_noembed(prov, href) };
        }
    }

    return nullopt;
}

optional<Node> parse_url(string_view word)
{
    if (auto n = parse_embeds(word)) {
        return n;
    }

    const static string allowed_prefixes[5]
        = { "http:", "https:", "ftp:", "ftps:", "bitcoin:" };

    bool valid = false;
    for (auto& pre : allowed_prefixes) {
        if (std::equal(pre.begin(), pre.end(), word.begin())) {
            valid = true;
            break;
        }
    }
    if (!valid || !validate_url(word)) {
        return nullopt;
    }
    // Don't open a new tab for magnet links
    return { render_link(word, word, word[0] != 'm') };
}
