#include "../util.hh"
#include "../lang.hh"
#include "../options/options.hh"
#include "../state.hh"
#include "models.hh"
#include <ctime>
#include <string>
#include <tuple>

// Renders "56 minutes ago" or "in 56 minutes" like relative time text
static std::string ago(
    time_t n, const std::tuple<std::string, std::string>& units, bool is_future)
{
    auto count = pluralize(n, units);
    return is_future ? lang->posts.at("in") + " " + count
                     : count + " " + lang->posts.at("ago");
}

std::string relative_time(time_t then)
{
    auto now = (float)std::time(0);
    auto t = (now - (float)then) / 60;
    auto is_future = false;
    if (t < 1) {
        if (t > -5) { // Assume to be client clock imprecision
            return lang->posts.at("justNow");
        }
        is_future = true;
        t = -t;
    }

    const int divide[4] = { 60, 24, 30, 12 };
    const static std::string unit[4] = { "minute", "hour", "day", "month" };
    for (int i = 0; i < 4; i++) {
        if (t < divide[i]) {
            return ago(t, lang->plurals.at(unit[i]), is_future);
        }
        t /= divide[i];
    }

    return ago(t, lang->plurals.at("year"), is_future);
}

Node render_post_link(uint64_t id, const LinkData& data)
{
    const bool cross_thread = data.op != page->thread;
    const bool index_page = !page->thread && !page->catalog;
    const std::string id_str = std::to_string(id);

    std::ostringstream url;
    if (cross_thread || index_page) {
        url << "/all/" << id_str;
    }
    url << "#p" << id_str;

    std::ostringstream text;
    text << ">>" << id_str;
    if (cross_thread && !index_page) {
        text << " ➡";
    }
    if (post_ids->mine.count(id)) { // Post, the user made
        text << ' ' << lang->posts.at("you");
    }

    Node n = Node("em");
    n.children.push_back({
        "a",
        {
            { "class", "post-link" }, { "href", url.str() },
        },
        text.str(),
    });
    if (options->post_inline_expand) {
        n.children.push_back({
            "a",
            {
                { "class", "hash-link" }, { "href", url.str() },
            },
            " #",
        });
    }

    // Inline linked-to post
    if (data.is_inlined) {
        n.children.push_back(posts->at(id).render());
    }

    return n;
}
