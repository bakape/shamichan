#pragma once

#include <string>

// Client-side options
class Options {
public:
    bool hide_thumbs = false, image_hover = true, webm_hover = false,
         notification = true, anonymise = false, post_inline_expand = true,
         relative_time = false, now_playing = false, illya_dance = false,
         illya_dance_mute = false, horizontal_posting = false,
         hide_recursively = false, work_mode_toggle = false, user_BG = false,
         custom_css_toggle = false, mascot = false, always_lock = false;
    unsigned int new_post = 78, toggle_spoiler = 73, done = 83, expand_all = 69,
                 work_mode = 66;
    enum class FittingMode {
        none,
        width,
        screen
    } inline_fit
        = FittingMode::width;
    std::string theme = "moe", // TODO: Read from configs
        custom_css = "";

    // Read options from memory and/or load defaults, where needed
    Options() { load(); }

    // Load properties from localStorage
    void load();

private:
    // Load a boolean property from localStorage, or a default
    void load_bool(bool& val, const std::string& key);

    // Load a uint property from localStorage, or a default
    void load_uint(unsigned int& val, const std::string& key);

    // Load a string property from localStorage, or a default
    void load_string(std::string& val, const std::string& key);
};

// Client-side options
extern Options* options;
