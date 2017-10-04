#pragma once

#include <string>
#include <unordered_map>

using std::string;
using std::unordered_map;

// Public server-wide global configurations
class Config {
public:
    bool captcha, mature, disable_user_boards, prune_threads;
    unsigned int thread_expiry_min, thread_expiry_max;
    string default_lang, default_css, image_root_override;
    unordered_map<string, string> links;

    // Parse JSON string
    Config(const string&);
};

// Server-wide global configuration, that affects the client
extern Config* config;

// Public board-specific configurations
class BoardConfig {
public:
    bool read_only, text_only, forced_anon;
    string title, notice, rules;

    // Parse JSON string
    BoardConfig(const string&);
};

// Public board-specific configurations
extern BoardConfig* board_config;

// Describes the current page
class Page {
public:
    bool catalog;
    unsigned int last_n, page;
    uint64_t thread;
    string board;

    // Detect the current page, by reading the current URL
    void detect();

private:
    // Find a numeric query parameter and parse it.
    // Returns 0, if none found.
    unsigned int find_query_param(const string& query, const string& param);
};

// Describes the current page
extern Page* page;

// Load initial application state
void load_state();
