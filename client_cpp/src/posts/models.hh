#pragma once

#include "../json.hh"
#include <map>
#include <optional>
#include <stdint.h>
#include <string>
#include <tuple>
#include <vector>

// Possible file types of a post image or thumbnail
enum class FileType : uint8_t {
    jpg,
    png,
    gif,
    webm,
    pdf,
    svg,
    mp4,
    mp3,
    ogg,
    zip,
    _7z,
    targz,
    tarxz,
    flac,
    noFile,
    txt,
};

class Image {
public:
    bool apng, audio, video, spoiler,
        large, // Render larger thumbnails
        expanded, // Thumbnail is expanded
        taller_than_viewport, // Image is taller than the current viewport
        revealed; // Revealing a hidden image with [Show]
    FileType file_type, thumb_type;
    uint16_t dims[4];
    uint32_t length;
    uint64_t size;
    std::string artist, title, MD5, SHA1, name;

    Image() {}

    // Parse from JSON
    Image(nlohmann::json& j);
};

// State of a post's text. Used for adding enclosing tags to the HTML while
// parsing.
struct TextState {
    bool spoiler, quote, code, have_syncwatch;
    int newlines, dice_index; // Index of the next dice array item to use
};

// Single hash command result delivered from the server
class Command {
public:
    enum class Type { dice, flip, eight_ball, sync_watch, pyu, pcount } typ;
    union {
        bool flip;
        uint64_t pyu, pcount;
        uint64_t sync_watch[5];
        std::string* eight_ball;
        std::vector<uint16_t>* dice;
    };

    Command() {}

    // Parse from JSON
    Command(nlohmann::json& j);

    ~Command();
};

// Generic post model
class Post {
public:
    bool editing, deleted, sage, banned, sticky, locked, seenOnce, hidden;
    std::optional<Image> image;
    uint64_t id, op, time;
    std::string body, board;
    std::optional<std::string> name, trip, auth, subject, flag, poster_id;
    TextState state;
    std::vector<Command> commands;
    std::map<uint64_t, uint64_t> backlinks;
    std::vector<std::tuple<uint64_t, uint64_t>> links;

    Post() {}

    // Parse from JSON
    Post(nlohmann::json& j);
};

// Contains thread metadata
class Thread {
public:
    uint64_t post_ctr, image_ctr, reply_time, bump_time;
};
