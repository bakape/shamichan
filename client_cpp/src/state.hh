#pragma once

#include "posts/models.hh"
#include "util.hh"
#include <map>
#include <nlohmann/json.hpp>
#include <string>
#include <unordered_map>
#include <unordered_set>

// Contains all posts currently loaded on the page. Posts might or might not
// be actually displayed.
inline std::map<unsigned long, Post> posts;

// Caches the origin of the page
inline std::string location_origin;

// Loaded thread metadata
inline std::unordered_map<unsigned long, Thread> threads;

// Debug mode. Can be enabled by setting the "debug=true" query string.
inline bool debug = false;

// Public server-wide global configurations
class Config {
public:
    bool captcha, mature, disable_user_boards, prune_threads;
    unsigned thread_expiry_min, thread_expiry_max;
    std::string default_lang, default_css, image_root_override;
    std::unordered_map<std::string, std::string> links;

    Config() {}

    // Parse JSON string
    Config(const c_string_view&);
};

// Server-wide global configuration, that affects the client
inline Config config;

// Public board-specific configurations
class BoardConfig {
public:
    bool read_only, text_only, forced_anon, rb_text, pyu;
    std::string title, notice, rules;

    // Banner file types
    std::vector<FileType> banners;

    BoardConfig() {}

    // Parse from JSON
    BoardConfig(nlohmann::json&&);
};

// Public board-specific configurations
inline BoardConfig board_config;

// Map of all existing boards to their titles
inline std::map<std::string, std::string> boards;

// Describes the current page
class Page {
public:
    bool catalog = false, last_100 = false;
    unsigned page = 0, page_total = 0;
    unsigned long thread = 0, post = 0;
    std::string board;

    Page() {}

    Page(const std::string&);
};

// Describes the current page
inline Page page;

// Load initial application state
void load_state();

// Load posts from inlined JSON. Returns a vector of detected thread IDs.
// TODO: Fetch this as binary data from the server. It is probably a good idea
// to do this and configuration fetches in one request.
void load_posts(std::string_view data);

// Stores post ID of various catagories
struct PostIDs {
    std::unordered_set<unsigned long> mine, // Post, the user has created
        seen_replies, // Replies to the user's posts, the user has seen
        seen_posts, // Posts the user has seen
        hidden; // Posts the user has hidden
};

inline PostIDs post_ids;

// Types of post ID storage in the database
enum class StorageType : int { mine, seen_replies, seen_posts, hidden };

// Used to decode thread JSON
// TODO: Get rid of this in favour of a binary decoder
class ThreadDecoder : public Thread {
public:
    std::vector<Post> posts;

    // Parse from JSON
    ThreadDecoder(nlohmann::json& j);
};
