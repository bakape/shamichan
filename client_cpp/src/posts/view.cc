#include "view.hh"
#include "../lang.hh"
#include "../options/main.hh"
#include "../state.hh"
#include "../util.hh"
#include "countries.hh"
#include "models.hh"
#include <iomanip>
#include <iostream>
#include <sstream>
#include <tuple>

using brunhild::Children;

Node PostView::render(const Post& p)
{
    brunhild::Attrs attrs;
    attrs["id"] = "#p" + std::to_string(p.id);
    attrs["class"] = "glass";
    if (p.editing) {
        attrs["class"] += " editing";
    }

    Children ch;
    ch.reserve(5);
    if (p.deleted) {
        attrs["class"] += " deleted";
        ch.push_back({ "input",
            {
                { "type", "checkbox" }, { "class", "deleted-toggle" },
            } });
    }
    ch.push_back(render_header(p));

    return Node("article", attrs, ch);
}

Node PostView::render_header(const Post& p)
{
    Children ch;
    ch.reserve(8);

    // TODO: Check if staff, and render moderator checkbox

    if (p.sticky) {
        ch.push_back({ "svg",
            {
                { "xmlns", "http://www.w3.org/2000/svg" }, { "width", "8" },
                { "height", "8" }, { "viewBox", "0 0 8 8" },
            },
            R"'(<path d="M1.34 0a.5.5 0 0 0 .16 1h.5v2h-1c-.55 0-1 .45-1 1h3v3l.44 1 .56-1v-3h3c0-.55-.45-1-1-1h-1v-2h.5a.5.5 0 1 0 0-1h-4a.5.5 0 0 0-.09 0 .5.5 0 0 0-.06 0z" />)'" });
    }
    if (p.locked) {
        ch.push_back({ "svg",
            {
                { "xmlns", "http://www.w3.org/2000/svg" }, { "width", "8" },
                { "height", "8" }, { "viewBox", "0 0 8 8" },
            },
            R"'(<path d="M3 0c-1.1 0-2 .9-2 2v1h-1v4h6v-4h-1v-1c0-1.1-.9-2-2-2zm0 1c.56 0 1 .44 1 1v1h-2v-1c0-.56.44-1 1-1z" transform="translate(1)" />)'" });
    }

    if (p.subject) {
        std::string s;
        s.reserve(p.subject->size() + 6); // +2 unicode chars
        s = "「" + *p.subject + "」";
        ch.push_back({ "h3", s, true });
    }
    ch.push_back(render_name(p));
    if (p.flag) {
        auto& flag = *p.flag;
        ch.push_back({ "img",
            {
                { "class", "flag" },
                { "src", "/assets/flags/" + flag + ".svg" },
                { "title", countries.count(flag) ? countries.at(flag) : flag },
            } });
    }
    ch.push_back(render_time(p.time));

    const auto id_str = std::to_string(p.id);
    std::string url = "#p" + id_str;
    if (!page->thread && !page->catalog) {
        url = "/all/" + id_str + "?last=100" + url;
    }
    ch.push_back({ "nav", {},
        {
            { "a",
                {
                    { "href", url },
                },
                "No." },
            { "a",
                {
                    { "class", "quote" }, { "href", url },
                },
                id_str },
        } });
    ch.push_back({ "a", { { "class", "control" } },
        {
            { "svg",
                {
                    { "xmlns", "http://www.w3.org/2000/svg" }, { "width", "8" },
                    { "height", "8" }, { "viewBox", "0 0 8 8" },
                },
                R"'(<path d="M1.5 0l-1.5 1.5 4 4 4-4-1.5-1.5-2.5 2.5-2.5-2.5z" transform="translate(0 1)" />)'" },
        } });

    return Node("header", { { "class", "spaced" } }, ch);
}

Node PostView::render_name(const Post& p)
{
    Node n("b",
        {
            { "class", "name spaced" },
        });
    n.children.reserve(5);
    if (p.sage) {
        n.attrs["class"] += " sage";
    }

    if (options->anonymise) {
        n.children = { Node("span", lang->posts.at("anon")) };
        return n;
    }

    if (p.name || !p.trip) {
        n.children.push_back(p.name ? Node("span", *p.name, true)
                                    : Node("span", lang->posts.at("anon")));
    }
    if (p.trip) {
        n.children.push_back({ "code", "!" + *p.trip, true });
    }
    if (p.poster_id) {
        n.children.push_back({ "span", *p.poster_id, true });
    }
    if (p.auth) {
        n.attrs["class"] += " admin";
        n.children.push_back({ "span", "## " + lang->posts.at(*p.auth) });
    }
    if (post_ids->mine.count(p.id)) {
        n.children.push_back({ "i", lang->posts.at("you") });
    }

    return n;
}

Node PostView::render_time(time_t time)
{
    auto then = std::localtime(&time);
    std::ostringstream s;
    s << std::setfill('0') << std::setw(2);

    // Renders classic absolute timestamp
    s << then->tm_mday << ' ' << lang->calendar[then->tm_mon] << ' '
      << 1900 + then->tm_year << " (" << lang->week[then->tm_wday] << ") "
      << then->tm_hour << ':' << then->tm_min;
    const auto abs = s.str();

    const auto rel = relative_time(time);

    return Node("time",
        {
            { "title", options->relative_time ? abs : rel },
        },
        options->relative_time ? rel : abs);
}

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
    const string unit[4] = { "minute", "hour", "day", "month" };
    for (int i = 0; i < 4; i++) {
        if (t < divide[i]) {
            return ago(t, lang->plurals.at(unit[i]), is_future);
        }
        t /= divide[i];
    }

    return ago(t, lang->plurals.at("year"), is_future);
}
