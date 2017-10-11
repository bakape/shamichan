#pragma once

#include "../json.hh"
#include <map>
#include <optional>
#include <stdint.h>
#include <string>
#include <unordered_map>
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
    std::optional<bool> apng, // PNG image is APNG
        audio, // Has audio
        video, // Has video
        spoiler, // Is spoilered
        large, // Render larger thumbnails
        expanded, // Thumbnail is expanded
        taller_than_viewport, // Image is taller than the viewport
        revealed; // Revealing a hidden image with [Show]
    FileType file_type, // File type of source file
        thumb_type; // File type of thumbnail
    uint16_t dims[4];
    std::optional<uint32_t> length; // Length of media, if a media file
    uint64_t size;
    std::optional<std::string> artist, // Media file artist meta info
        title; // Media file title meta info
    std::string MD5, // MD5 hash of source file
        SHA1, // SHA1 hash of source file
        name; // Name the file was uploaded with

    // Parse from JSON
    Image(nlohmann::json&);
};

// State of a post's text. Used for adding enclosing tags to the HTML while
// parsing.
struct TextState {
    bool spoiler = false, // Current text is spoilered
        quote = false, // Current line is spoilered
        code = false, // Text is inside code block
        have_syncwatch = false; // Text contains #syncwatch command(s)
    int newlines = 0, // Number of newlines in text
        dice_index = 0; // Index of the next dice array item to use
};

// Single hash command result delivered from the server
class Command {
public:
    // Indicates the contained type
    enum class Type { dice, flip, eight_ball, sync_watch, pyu, pcount } typ;

    // Use typ, to get out the relevant value
    bool flip; // Result of flip command
    uint64_t count; // Somekind of counter result
    uint64_t sync_watch[5]; // Syncwatch parameters
    std::vector<uint16_t> dice; // Result of dice throw
    std::string eight_ball; // Result of #8ball command

    // Parse from JSON
    Command(nlohmann::json&);
};

// Data associated with link to another post. Is always pared in a map with
// the ID of the linked post as a key.
struct LinkData {
    // The post and its subtree is now a child of the link
    bool is_inlined = false;
    // Parent thread ID of the post
    uint64_t op;
};

// Generic post model
class Post {
public:
    std::optional<bool> editing, // Post is currrently being edited
        deleted, // Deleted by moderator
        sage, // Poster disabled bumping of the parent thread
        banned, // Banned for this post by moderator
        sticky, // Thread is stickied. Only for OPs.
        locked, // Thread is locked. Only for OPs.
        seen, // The user has already seen this post
        hidden; // The post has been hidden by the user
    std::optional<Image> image;
    uint64_t id, op, time;
    std::string body, board;
    std::optional<std::string> name, // Name of poster
        trip, // Trip code of poster
        auth, // Staff title of poster
        subject, // Subject of thread. Only for OPs.
        flag, // Country code of poster
        poster_id; // Thread-level poster ID
    TextState state;
    std::vector<Command> commands; // Results of hash commands
    std::map<uint64_t, LinkData> backlinks; // Posts linking to this post
    std::unordered_map<uint64_t, LinkData> links; // Posts linked by this post

    // Parse from JSON
    Post(nlohmann::json&);

    // Required to place Post into collections
    Post() {}
};

// Contains thread metadata
class Thread {
public:
    uint64_t post_ctr, // Number of posts in thread
        image_ctr, // Number of images in thread
        reply_time, // Unix timestamp of last reply
        bump_time; // Unix timestamp of last bump
};
