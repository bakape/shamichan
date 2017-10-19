#pragma once

#include <string>

// Client-side options
class Options {
public:
    bool hide_thumbs = false, // Hide file thumbnails
        image_hover = true, // Preview images on hover
        webm_hover = false, // Preview videos on hover
        notification = true, // Display quote notifications, when tab is hidden
        anonymise = false, // Hide all poster names and trips
        post_inline_expand = true, // Expand post links inline on click
        relative_time = false, // Display timestamps relative to now
        now_playing = false, // Show r/a/dio Now Playing info in top banner
        illya_dance = false, // Dancing loli in background
        illya_dance_mute = false, // Mute dancing loli
        horizontal_posting = false, // Arrange posts in a flexbox
        hide_recursively = false, // Hide posts, that quote a hidden post
        work_mode_toggle = false, // Work mode AKA Boss mode
        user_BG = false, // Show custom user-set background
        custom_css_toggle = false, // Enable user-ser CSS
        mascot = false, // Show user-set mascot
        always_lock = false; // Lock to thread bottom, even when tab hidden

    // Reverse image search engines
    bool google = true, iqdb = false, sauce_nao = true, what_anime = false,
         desu_storage = false, exhentai = false;

    // Keybinding
    unsigned int new_post = 78, // Create new post
        toggle_spoiler = 73, // Toggle image spoiler
        done = 83, // Close post
        expand_all = 69, // Expand all images
        work_mode = 66; // Toggle Work AKA Boss mode

    // Fitting mode for image expansion
    enum class FittingMode {
        width, // Fit to width
        screen // Fit to screen
    } inline_fit
        = FittingMode::width;
    std::string theme = "moe", // CSS theme; TODO: Read default from configs
        custom_css = ""; // Custom user-set CSS

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
