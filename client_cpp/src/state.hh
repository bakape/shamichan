#pragma once

#include <string>
#include <unordered_map>

using std::string;
using std::unordered_map;

// Server-wide global configurations
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
