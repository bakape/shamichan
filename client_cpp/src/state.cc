#include "state.hh"
#include "lang.hh"
#include "options/options.hh"
#include "page/page.hh"
#include "posts/models.hh"
#include "util.hh"
#include <array>
#include <emscripten.h>
#include <emscripten/bind.h>
#include <map>
#include <string_view>
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

// Places inverse post links into backlinks for later assignment to individual
// post models
static void extract_backlinks(const Post& p, Backlinks& backlinks)
{
    for (auto&& [target_id, _] : p.links) {
        backlinks[target_id][p.id] = { false, p.op, p.board };
    }
}

// Extract thread data from JSON and populate post collection.
// Places inverse post links into backlinks for later assignment to individual
// post models.
static void extract_thread(json& j, Backlinks& backlinks)
{
    // TODO: Homogenize board and thread page data structure
    auto thread = ThreadDecoder(j);
    auto op = page.thread ? thread.posts[0] : Post(j);
    const string board = thread.board;
    const unsigned long thread_id = op.id;
    op.op = thread_id;
    op.board = board;
    extract_backlinks(op, backlinks);
    (threads)[thread_id] = static_cast<Thread>(thread);
    (posts)[thread_id] = std::move(op);

    for (auto post : thread.posts) {
        post.board = board;
        post.op = thread_id;
        extract_backlinks(post, backlinks);
        (posts)[post.id] = post;
    }
}

void load_posts(std::string_view data)
{
    Backlinks backlinks;
    backlinks.reserve(128);
    auto j = json::parse(data);
    if (page.thread) {
        extract_thread(j, backlinks);
    } else {
        page.page_total = j["pages"];
        for (auto& thread : j["threads"]) {
            extract_thread(thread, backlinks);
        }

        // TODO: Catalog pages
    }

    // Assign backlinks to their post models
    for (auto [target_id, data] : backlinks) {
        if (posts.count(target_id)) {
            posts.at(target_id).backlinks = std::move(data);
        }
    }
}

void load_state()
{
    // Order is important to prevent race conditions

    debug = val::global("location")["search"].as<string>().find("debug=true")
        != string::npos;
    auto location = val::global("location");
    location_origin = location["origin"].as<string>();
    page = { location["href"].as<string>().substr(location_origin.size()) };
    options.load();
    lang.load();

    for (auto& pair : json::parse(get_inner_html("board-title-data"))) {
        boards[pair["id"]] = pair["title"];
    }

    config = { get_inner_html("conf-data") };
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

BoardConfig::BoardConfig(nlohmann::json&& j)
{
    read_only = j["readOnly"];
    text_only = j["textOnly"];
    forced_anon = j["forcedAnon"];
    rb_text = j["rbText"];
    pyu = j["pyu"];
    title = j["title"];
    rules = j["rules"];
    notice = j["notice"];

    auto& b = j["banners"];
    banners.reserve(b.size());
    for (auto& type : b) {
        banners.push_back(static_cast<FileType>(type));
    }
}

// Parse string_view to unsigned int. Invalid string returns 0.
static unsigned parse_uint(const std::string_view s)
{
    static constexpr std::array<unsigned, 20> pow10 = []() {
        std::array<unsigned, 20> arr = { { 0 } };
        arr[0] = 1;
        for (unsigned i = 1; i < 20; i++) {
            arr[i] = arr[i - 1] * 10;
        }
        return arr;
    }();

    const size_t size = s.size();
    if (size > 20) {
        return 0;
    }

    unsigned result = 0;
    for (size_t i = 0; i < size; i++) {
        const char ch = s[i];
        if (ch < '0' || ch > '9') {
            return 0;
        }
        result += pow10[size - i - 1] * (ch - '0');
    }
    return result;
}

// Find a numeric query parameter and parse it.
// Returns 0, if none found.
static unsigned find_query_param(std::string_view query, const char* param)
{
    size_t i = query.find(param);
    if (i == string::npos) {
        return 0;
    }
    i += strlen(param) + 1;
    return parse_uint(query.substr(i, query.find_first_of('&', i)));
}

Page::Page(const string& href)
{
    const auto i_query = href.find('?');
    const auto i_hash = href.find('#');
    const auto view = std::string_view(href);

    // Parse the path URL
    size_t i = string::npos;
    if (i_hash != string::npos) {
        i = i_hash;
    }
    if (i_query != string::npos) {
        i = i_query;
    }
    const auto path = view.substr(0, i);
    i = path.find_first_of('/', 1);
    board = path.substr(1, i - 1);
    if (i != path.size() - 1) {
        const auto thread_str = path.substr(i + 1, -1);
        if (thread_str == "catalog") {
            catalog = true;
        } else {
            thread = parse_uint(thread_str);
        }
    }

    // Parse query string
    if (i_query != string::npos) {
        const auto query = view.substr(i_query, i_hash);
        if (thread) {
            last_100 = find_query_param(query, "last") == 100;
        } else if (!catalog) {
            page = find_query_param(query, "page");
        }
    }

    // Parse hash
    if (i_hash != string::npos) {
        const auto hash = view.substr(i_hash);
        if (hash.size() >= 3) {
            post = parse_uint(hash.substr(2));
        }
    }
}

void add_to_storage(int typ, const std::vector<unsigned long> ids)
{
    std::unordered_set<unsigned long>* set = nullptr;
    switch (static_cast<StorageType>(typ)) {
    case StorageType::mine:
        set = &post_ids.mine;
        break;
    case StorageType::seen_posts:
        set = &post_ids.seen_posts;
        break;
    case StorageType::seen_replies:
        set = &post_ids.seen_replies;
        break;
    case StorageType::hidden:
        set = &post_ids.hidden;
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

    // Redundant field on thread pages
    id = page.thread ? page.thread : (unsigned long)(j["id"]);

    post_ctr = j["post_count"];
    image_ctr = j["image_count"];
    time = j["time"];
    reply_time = j["update_time"];
    bump_time = j["bump_time"];
    board = j["board"];
    subject = j["subject"];
    if (!page.catalog) {
        auto& p = j.at("posts");
        posts.reserve(p.size());
        for (auto& data : p) {
            posts.push_back(Post(data));
        }
    }
}
