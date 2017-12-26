#include "state.hh"
#include "connection/connection.hh"
#include "db.hh"
#include "lang.hh"
#include "options/options.hh"
#include "page/page.hh"
#include "posts/hide.hh"
#include "posts/models.hh"
#include "util.hh"
#include <emscripten.h>
#include <emscripten/bind.h>
#include <map>
#include <unordered_map>
#include <unordered_set>
#include <utility>

using emscripten::val;
using nlohmann::json;
using std::string;

// Inverse map of posts linking posts by post ID.
// <linked_post_id <linker_post_id, {false, linker_post_thread_id}>>
typedef std::unordered_map<unsigned long, std::map<unsigned long, LinkData>>
    Backlinks;

Config const* config;
BoardConfig const* board_config;
std::unordered_set<string> const* boards;
std::map<string, string> const* board_titles;

Page* page;
bool debug = false;
string const* location_origin;

PostIDs* post_ids;
std::map<unsigned long, Post>* posts;
std::unordered_map<unsigned long, Thread>* threads;

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
static void extract_thread(json& j, Backlinks& backlinks)
{
    // TODO: Homogenize board and thread page data structure
    auto thread = ThreadDecoder(j);
    auto op = page->thread ? thread.posts[0] : Post(j);
    const string board = thread.board;
    const unsigned long thread_id = op.id;
    op.op = thread_id;
    op.board = board;
    extract_backlinks(op, backlinks);
    (*threads)[thread_id] = static_cast<Thread>(thread);
    (*posts)[thread_id] = std::move(op);

    for (auto post : thread.posts) {
        post.board = board;
        post.op = thread_id;
        extract_backlinks(post, backlinks);
        (*posts)[post.id] = post;
    }
}

void load_posts(std::string_view data)
{
    Backlinks backlinks;
    backlinks.reserve(128);
    auto j = json::parse(data);
    if (page->thread) {
        extract_thread(j, backlinks);
    } else {
        page->page_total = j["pages"];
        for (auto& thread : j["threads"]) {
            extract_thread(thread, backlinks);
        }

        // TODO: Catalog pages
    }

    // Assign backlinks to their post models
    for (auto[target_id, data] : backlinks) {
        if (posts->count(target_id)) {
            posts->at(target_id).backlinks = std::move(data);
        }
    }

    recurse_hidden_posts();
}

void load_state()
{
    // Order is important to prevent race conditions after the database is
    // loaded

    debug = val::global("location")["search"].as<string>().find("debug=true")
        != -1;
    page = new Page();
    page->detect();
    options = new Options();
    options->load();
    lang = new LanguagePack();

    location_origin
        = new string(val::global("location")["origin"].as<string>());

    std::map<string, string> titles;
    for (auto& pair : json::parse(get_inner_html("board-title-data"))) {
        titles[pair["id"]] = pair["title"];
    }
    board_titles = new std::map<string, string>(titles);

    // TODO: This should be read from a concurrent server fetch

    config = new Config(c_string_view((char*)EM_ASM_INT_V({
        var s = JSON.stringify(window.config);
        var len = lengthBytesUTF8(s) + 1;
        var buf = Module._malloc(len);
        stringToUTF8(s, buf, len);
        return buf;
    })));

    std::unordered_set<string> b_temp
        = json::parse(c_string_view((char*)EM_ASM_INT_V({
              var s = JSON.stringify(window.boards);
              var len = lengthBytesUTF8(s) + 1;
              var buf = Module._malloc(len);
              stringToUTF8(s, buf, len);
              return buf;
          })));
    boards = new std::unordered_set<string>(b_temp);

    board_config = new BoardConfig(c_string_view((char*)EM_ASM_INT_V({
        var s = document.getElementById('board-configs').innerHTML;
        var len = lengthBytesUTF8(s) + 1;
        var buf = Module._malloc(len);
        stringToUTF8(s, buf, len);
        return buf;
    })));

    posts = new std::map<unsigned long, Post>();
    post_ids = new PostIDs{};
    threads = new std::unordered_map<unsigned long, Thread>();
    init_connectivity();
    auto wg = new WaitGroup(2, &load_post_ids);
    open_db(wg);
    if (page->thread) {
        conn_SM->feed(ConnEvent::start);
        conn_SM->once(ConnState::synced, [=]() { wg->done(); });
    } else {
        // TODO: Do this with an XHR
        const c_string_view data = get_inner_html("post-data");
        load_posts(static_cast<std::string_view>(data));
        wg->done();
    }
}

Config::Config(const c_string_view& s)
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

BoardConfig::BoardConfig(const c_string_view& s)
{
    auto j = json::parse(s);

    read_only = j["readOnly"];
    text_only = j["textOnly"];
    forced_anon = j["forcedAnon"];
    non_live = j["nonLive"];
    title = j["title"];
    rules = j["rules"];
    notice = j["notice"];

    auto& b = j["banners"];
    banners.reserve(b.size());
    for (auto& type : b) {
        banners.push_back(static_cast<FileType>(type));
    }
}

void Page::detect()
{
    // This needs to be parsed from the board data, if any
    page_total = 0;

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

void add_to_storage(int typ, const std::vector<unsigned long> ids)
{
    std::unordered_set<unsigned long>* set = nullptr;
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
// Decode a key, that may not be in the object
#define OPT_DECODE(key)                                                        \
    if (j.count(#key)) {                                                       \
        key = j[#key];                                                         \
    }

    OPT_DECODE(deleted)
    OPT_DECODE(locked)
    OPT_DECODE(sticky)
    if (j.count("nonLive")) {
        non_live = j["nonLive"];
    }

    // Redundant field on thread pages
    id = page->thread ? page->thread : (unsigned long)(j["id"]);

    post_ctr = j["postCtr"];
    image_ctr = j["imageCtr"];
    time = j["time"];
    reply_time = j["replyTime"];
    bump_time = j["bumpTime"];
    board = j["board"];
    if (!page->catalog) {
        auto& p = j.at("posts");
        posts.reserve(p.size());
        for (auto& data : p) {
            posts.push_back(Post(data));
        }
    }
}
