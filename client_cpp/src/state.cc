#include "state.hh"
#include "json.hh"
#include <emscripten.h>
#include <emscripten/bind.h>

using json = nlohmann::json;
using emscripten::val;

Config* config = nullptr;
BoardConfig* board_config = nullptr;
Page* page = nullptr;

void load_state()
{
    // TODO: This should be read from a concurrent server fetch
    const char* conf = (char*)EM_ASM_INT_V({
        var s = JSON.stringify(window.config);
        var len = s.length + 1;
        var buf = Module._malloc(len);
        stringToUTF8(s, buf, len);
        return buf;
    });
    config = new Config(string(conf));
    delete[] conf;

    const char* board_conf = (char*)EM_ASM_INT_V({
        var s = document.getElementById('board-configs').innerHTML;
        var len = s.length + 1;
        var buf = Module._malloc(len);
        stringToUTF8(s, buf, len);
        return buf;
    });
    board_config = new BoardConfig(string(board_conf));
    delete[] board_conf;

    page = new Page();
    page->detect();
}

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

BoardConfig::BoardConfig(const string& s)
{
    auto j = json::parse(s);

    read_only = j["readOnly"];
    text_only = j["textOnly"];
    forced_anon = j["forcedAnon"];
    title = j["title"];
    rules = j["rules"];
    notice = j["notice"];
}

void Page::detect()
{
    val location = val::global("location");
    const string path = location["pathname"].as<string>();
    const string query = location["search"].as<string>();

    // Parse the path URL
    size_t i = path.find_first_of('/', 1);
    board = path.substr(1, i - 1);
    if (i != path.size() - 1) {
        const string thread_str = path.substr(i + 1, -1);
        if (thread_str == "catalog") {
            catalog = true;
        } else {
            thread = std::stoul(thread_str);
        }
    }

    // Parse query string
    if (query != "") {
        if (thread) {
            last_n = find_query_param(query, "last");
        } else if (!catalog) {
            page = find_query_param(query, "page");
        }
    }
}

unsigned int Page::find_query_param(const string& query, const string& param)
{
    size_t i = query.find(param);
    if (i == -1) {
        return 0;
    }
    i += param.size() + 1;
    const string s = query.substr(i, query.find_first_of('&', i));
    return std::stoul(s);
}
