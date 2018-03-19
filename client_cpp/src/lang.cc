#include "lang.hh"
#include "util.hh"
#include <nlohmann/json.hpp>

LanguagePack lang;

using nlohmann::json;

void LanguagePack::load()
{
    auto j = json::parse(get_inner_html("lang-data"));
    auto& t = j["time"];

    load_map(posts, j["posts"]);
    load_map(ui, j["ui"]);
    load_tuple_map(plurals, j["plurals"]);
    load_tuple_map(forms, j["forms"]);
    load_array(calendar, t["calendar"]);
    load_array(week, t["week"]);
    load_array(sync, j["sync"]);
}

template <class T> void LanguagePack::load_array(T& arr, json& j)
{
    for (size_t i = 0; i < std::extent<T>::value; i++) {
        arr[i] = j[i];
    }
}

void LanguagePack::load_map(LanguagePack::StringMap& m, json& j)
{
    m.reserve(j.size());
    for (json::iterator it = j.begin(); it != j.end(); it++) {
        m[it.key()] = it.value();
    }
}

void LanguagePack::load_tuple_map(LanguagePack::TupleMap& map, json& j)
{
    map.reserve(j.size());
    for (json::iterator it = j.begin(); it != j.end(); it++) {
        auto const& val = it.value();
        map[it.key()]
            = { val[0].get<std::string>(), val[1].get<std::string>() };
    }
}
