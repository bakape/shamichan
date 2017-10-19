#include "options.hh"
#include "../local_storage.hh"

// TODO: Expose Options::load() to JS side for modification loading

Options* options = nullptr;

// TODO: Implement observer pattern. We don't actually need unregistering
// though and can use function pointers only as observers.

void Options::load()
{
    load_bool(hide_thumbs, "hideThumbs");
    load_bool(image_hover, "imageHover");
    load_bool(webm_hover, "webmHover");
    load_bool(notification, "notification");
    load_bool(anonymise, "anonymise");
    load_bool(post_inline_expand, "postInlineExpand");
    load_bool(relative_time, "relativeTime");
    load_bool(now_playing, "nowPlaying");
    load_bool(illya_dance, "illyaDance");
    load_bool(illya_dance_mute, "illyaDanceMute");
    load_bool(horizontal_posting, "horizontalPosting");
    load_bool(hide_recursively, "hideRecursively");
    load_bool(work_mode_toggle, "workModeToggle");
    load_bool(user_BG, "userBG");
    load_bool(custom_css_toggle, "customCSS");
    load_bool(mascot, "mascot");
    load_bool(always_lock, "alwaysLock");
    load_bool(google, "google");
    load_bool(iqdb, "iqdb");
    load_bool(sauce_nao, "saucenao");
    load_bool(what_anime, "whatAnime");
    load_bool(desu_storage, "desustorage");
    load_bool(exhentai, "exhentai");

    load_uint(new_post, "newPost");
    load_uint(toggle_spoiler, "toggleSpoiler");
    load_uint(done, "done");
    load_uint(expand_all, "expandAll");
    load_uint(work_mode, "workMode");

    if (auto const s = local_storage_get("inlineFit"); s == "width") {
        inline_fit = FittingMode::width;
    } else if (s == "screen") {
        inline_fit = FittingMode::screen;
    }

    load_string(theme, "theme");
    load_string(custom_css, "customCSS");
}

void Options::load_bool(bool& val, const std::string& key)
{
    auto const s = local_storage_get(key);
    if (s != "") {
        val = s == "true";
    }
}

void Options::load_uint(unsigned int& val, const std::string& key)
{
    auto const s = local_storage_get(key);
    if (s != "") {
        val = std::stoul(s);
    }
}

void Options::load_string(std::string& val, const std::string& key)
{
    auto s = local_storage_get(key);
    if (s != "") {
        val = s;
    }
}
