#include "url.hh"
#include "etc.hh"
#include <algorithm>
#include <emscripten.h>

using std::string_view;
using std::string;
using std::optional;
using std::nullopt;

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

optional<brunhild::Node> parse_url(string_view word)
{
    // TODO: parse embeds

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
