#include "lang.hh"
#include "json.hh"
#include "util.hh"

LanguagePack const* lang = nullptr;

using json = nlohmann::json;

LanguagePack::LanguagePack()
{
    auto j = json::parse(get_inner_html("lang-data"));

    load_map(posts, j["posts"]);
    load_map(ui, j["ui"]);

    auto& pl = j["plurals"];
    plurals.reserve(pl.size());
    for (json::iterator it = pl.begin(); it != pl.end(); ++it) {
        auto const& val = it.value();
        plurals[it.key()]
            = { val[0].get<std::string>(), val[1].get<std::string>() };
    }

    auto& t = j["time"];
    load_array(calendar, t["calendar"]);
    load_array(week, t["week"]);
    load_array(sync, j["sync"]);
}

void LanguagePack::load_map(
    std::unordered_map<std::string, std::string>& m, nlohmann::json& j)
{
    m.reserve(j.size());
    for (json::iterator it = j.begin(); it != j.end(); ++it) {
        m[it.key()] = it.value();
    }
}
