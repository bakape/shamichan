#include "state.hh"
#include "db.hh"
#include "lang.hh"
#include "options/options.hh"
#include "posts/models.hh"
#include "util.hh"
#include <emscripten.h>
#include <emscripten/bind.h>
#include <map>
#include <unordered_map>
#include <unordered_set>
#include <utility>

using json = nlohmann::json;

// Inverse map of posts linking posts by post ID.
// <linked_post_id <linker_post_id, {false, linker_post_thread_id}>>
typedef std::unordered_map<uint64_t, std::map<uint64_t, LinkData>> Backlinks;

Config* config = nullptr;
BoardConfig* board_config = nullptr;
Page* page = nullptr;
PostIDs* post_ids = nullptr;
std::map<uint64_t, Post>* posts = nullptr;
string const* location_origin = nullptr;

// Places inverse post links into backlinks for later assignment to individual
// post models
static void extract_backlinks(const Post& p, Backlinks& backlinks)
{
    for (auto && [ target_id, _ ] : p.links) {
        backlinks[target_id][p.id] = { false, p.op };
    }
}

// Extract thread data from JSON and populate post collection.
// Places inverse post links into backlinks for later assignment to individual
// post models.
// Returns the id of the extracted thread;
static uint64_t extract_thread(json& j, Backlinks& backlinks)
{
    // TODO: Actually use the thread metadata
    auto thread = ThreadDecoder(j);

    const string board = j["board"];
    const uint64_t thread_id = j["id"];
    const auto op = Post(j);
    extract_backlinks(op, backlinks);
    (*posts)[thread_id] = op;

    for (auto post : thread.posts) {
        post.board = board;
        post.op = thread_id;
        extract_backlinks(post, backlinks);
        (*posts)[post.id] = post;
    }

    return thread_id;
}

// Load posts from inlined JSON. Returns a vector of detected thread IDs.
// TODO: Fetch this as binary data from the server. It is probably a good idea
// to do this and configuration fetches in one request.
static std::unordered_set<uint64_t> load_posts()
{
    Backlinks backlinks;
    backlinks.reserve(128);
    auto j = json::parse(get_inner_html("post-data"));
    auto thread_ids = std::unordered_set<uint64_t>();
    if (page->thread) {
        thread_ids.reserve(1);
        thread_ids.insert(extract_thread(j, backlinks));
    } else {
        thread_ids.reserve(15);
        for (auto& thread : j) {
            thread_ids.insert(extract_thread(thread, backlinks));
        }

        // TODO: Catalog pages
    }

    // Assign backlinks to their post models
    for (auto[target_id, data] : backlinks) {
        if (posts->count(target_id)) {
            posts->at(target_id).backlinks = std::move(data);
        }
    }

    return thread_ids;
}

void load_state()
{
    // Order is important to prevent race conditions after the database is
    // loaded

    page = new Page();
    page->detect();
    options = new Options();
    options->load();
    lang = new LanguagePack();

    location_origin = new string(
        emscripten::val::global("location")["origin"].as<string>());

    // TODO: This should be read from a concurrent server fetch
    config = new Config(convert_c_string(EM_ASM_INT_V({
        var s = JSON.stringify(window.config);
        var len = lengthBytesUTF8(s) + 1;
        var buf = Module._malloc(len);
        stringToUTF8(s, buf, len);
        return buf;
    })));

    board_config = new BoardConfig(convert_c_string(EM_ASM_INT_V({
        var s = document.getElementById('board-configs').innerHTML;
        var len = lengthBytesUTF8(s) + 1;
        var buf = Module._malloc(len);
        stringToUTF8(s, buf, len);
        return buf;
    })));

    posts = new std::map<uint64_t, Post>();
    post_ids = new PostIDs{};
    load_db(load_posts());
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

ThreadDecoder::ThreadDecoder(json& j)
{
    post_ctr = j["postCtr"];
    image_ctr = j["imageCtr"];
    reply_time = j["replyTime"];
    bump_time = j["bumpTime"];
    if (!page->catalog) {
        auto& p = j.at("posts");
        posts.reserve(p.size());
        for (auto& data : p) {
            posts.push_back(Post(data));
        }
    }
}
