#include "state.hh"
#include "../json.hh"

using json = nlohmann::json;

Config* config = nullptr;

Config::Config(const string& s)
{
    auto j = json::parse(s);

    captcha = j["captcha"];
    mature = j["mature"];
    disable_user_boards = j["disableUserBoards"];
    prune_threads = j["pruneThreads"];
    thread_expiry_min = j["threadExpiryMin"];
    thread_expiry_max = j["threadExpiryMax"];
    default_lang = j["defaultLang"];
    default_css = j["defaultCSS"];
    image_root_override = j["imageRootOverride"];

    auto& l = j["links"];
    for (json::iterator it = l.begin(); it != l.end(); ++it) {
        links[it.key()] = it.value();
    }
}
