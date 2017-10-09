#include "state.hh"
#include "db.hh"
#include "posts/models.hh"
#include "util.hh"
#include <emscripten.h>
#include <emscripten/bind.h>

using json = nlohmann::json;

Config* config = nullptr;
BoardConfig* board_config = nullptr;
Page* page = nullptr;
PostIDs* post_ids = nullptr;
std::unordered_map<uint64_t, Post>* posts = nullptr;

void load_state()
{
    page = new Page();
    page->detect();

    posts = new std::unordered_map<uint64_t, Post>();
    post_ids = new PostIDs{};
    load_db(load_posts());

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
    emscripten::val location = emscripten::val::global("location");
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

void add_to_storage(int typ, const std::vector<unsigned long> ids)
{
    std::unordered_set<uint64_t>* set = nullptr;
    switch (static_cast<StorageType>(typ)) {
    case StorageType::mine:
        set = &post_ids->mine;
        break;
    case StorageType::seen_posts:
        set = &post_ids->seen_posts;
        break;
    case StorageType::seen_replies:
        set = &post_ids->seen_replies;
        break;
    case StorageType::hidden:
        set = &post_ids->hidden;
        break;
    }
    set->reserve(set->size() + ids.size());
    set->insert(ids.begin(), ids.end());
}

EMSCRIPTEN_BINDINGS(module_state)
{
    emscripten::register_vector<unsigned long>("VectorUint64");
    emscripten::function("add_to_storage", &add_to_storage);
}

static std::vector<uint64_t> load_posts()
{
    auto j = json::parse(get_inner_html("post-data"));
    auto thread_ids = std::vector<uint64_t>(15);
    if (page->thread) {
        thread_ids.push_back(extract_thread(j));
    } else {
        for (auto& thread : j) {
            thread_ids.push_back(extract_thread(thread));
        }

        // TODO: Catalog pages
    }

    return thread_ids;
}

static uint64_t extract_thread(json& j)
{
    // TODO: Actually use the thread metadata
    auto thread = ThreadDecoder(j);
    posts->reserve(posts->size() + thread.posts.size() + 1);

    auto op = Post(j);
    const string board = op.board;
    const uint64_t thread_id = op.id;
    (*posts)[thread_id] = op;

    for (auto post : thread.posts) {
        post.board = board;
        post.op = thread_id;
        (*posts)[post.id] = post;
    }

    return thread_id;
}

ThreadDecoder::ThreadDecoder(json& j)
{
    post_ctr = j["postCtr"];
    image_ctr = j["imageCtr"];
    reply_time = j["replyTime"];
    bump_time = j["bumpTime"];
    if (page->catalog) {
        auto& p = j.at("posts");
        posts.reserve(p.size());
        for (auto& data : p) {
            posts.push_back(Post(data));
        }
    }
}
