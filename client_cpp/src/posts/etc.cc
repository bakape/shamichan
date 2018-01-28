#include "etc.hh"
#include "../lang.hh"
#include "../options/options.hh"
#include "../state.hh"
#include "../util.hh"
#include "models.hh"
#include <sstream>

using std::string;
using std::string_view;

// Renders "56 minutes ago" or "in 56 minutes" like relative time text
// Units is the index used to retrieve the language pack value for unit
// pluralization.
static string ago(time_t n, string units, bool is_future)
{
    auto count = pluralize(n, units);
    return is_future ? lang->posts.at("in") + " " + count
                     : count + " " + lang->posts.at("ago");
}

string relative_time(time_t then)
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
    const static string unit[4] = { "minute", "hour", "day", "month" };
    for (int i = 0; i < 4; i++) {
        if (t < divide[i]) {
            return ago(t, unit[i], is_future);
        }
        t /= divide[i];
    }

    return ago(t, "year", is_future);
}

Node render_post_link(unsigned long id, const LinkData& data)
{
    const bool cross_thread = data.op != page->thread;
    const bool index_page = !page->thread && !page->catalog;
    const string id_str = std::to_string(id);

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
    n.children.reserve(2);
    string cls = "post-link";
    if (post_ids->hidden.count(id)) {
        cls += " strikethrough";
    }
    n.children.push_back({ "a",
        {
            { "class", cls },
            { "href", url.str() },
        },
        text.str() });
    if (options->post_inline_expand) {
        n.children.push_back({ "a",
            {
                { "class", "hash-link" },
                { "href", url.str() },
            },
            " #" });
    }

    // Inline linked-to post
    if (data.is_inlined && posts->count(id)) {
        n.children.push_back(posts->at(id).render());
    }

    return n;
}

Node render_link(string_view url, string_view text, bool new_tab)
{
    Node n({
        "a",
        {
            { "rel", "noreferrer" },
            { "href", brunhild::escape(string(url)) },
        },
        string(text),
        true,
    });
    if (new_tab) {
        n.attrs["target"] = "_blank";
    }
    return n;
}

Post* match_post(const brunhild::Attrs& attrs)
{
    if (!attrs.count("data-id")) {
        return 0;
    }
    const unsigned long id = std::stoul(attrs.at("data-id"));
    if (!posts->count(id)) {
        return 0;
    }
    return &posts->at(id);
}
