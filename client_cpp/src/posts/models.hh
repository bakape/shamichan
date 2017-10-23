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
    no_file,
    txt,
};

// Maps file_type to the appropriate file extension
const std::unordered_map<FileType, std::string> file_extentions = {
    { FileType::jpg, "jpg" }, { FileType::png, "png" },
    { FileType::gif, "gif" }, { FileType::webm, "webm" },
    { FileType::pdf, "pdf" }, { FileType::svg, "svg" },
    { FileType::mp4, "mp4" }, { FileType::mp3, "mp3" },
    { FileType::ogg, "ogg" }, { FileType::zip, "zip" }, { FileType::_7z, "7z" },
    { FileType::targz, "tar.gz" }, { FileType::tarxz, "tar.xz" },
    { FileType::flac, "flac" }, { FileType::txt, "txt" },
};

class Image {
public:
    bool apng = false, // PNG image is APNG
        audio = false, // Has audio
        video = false, // Has video
        spoiler = false; // Is spoilered
    FileType file_type, // File type of source file
        thumb_type; // File type of thumbnail
    uint16_t dims[4];
    uint32_t length = 0; // Length of media, if a media file
    uint64_t size;
    std::optional<std::string> artist, // Media file artist meta info
        title; // Media file title meta info
    std::string MD5, // MD5 hash of source file
        SHA1, // SHA1 hash of source file
        name; // Name the file was uploaded with

    Image() = default;

    // Parse from JSON
    Image(nlohmann::json&);

    // Returns the path to this files's thumbnail
    std::string thumb_path() const;

    // Returns the path to the source file
    std::string source_path() const;

private:
    // Returns the root hosting address of all images
    std::string image_root() const;
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

// Forward declaration
class PostView;

// Generic post model
class Post {
public:
    bool editing = false, // Post is currrently being edited
        deleted = false, // Deleted by moderator
        sage = false, // Poster disabled bumping of the parent thread
        banned = false, // Banned for this post by moderator
        sticky = false, // Thread is stickied. Only for OPs.
        locked = false, // Thread is locked. Only for OPs.
        seen = false, // The user has already seen this post
        hidden = false; // The post has been hidden by the user
    std::optional<Image> image;
    uint64_t id, op;
    time_t time;
    std::string body, board;
    std::optional<std::string> name, // Name of poster
        trip, // Trip code of poster
        auth, // Staff title of poster
        subject, // Subject of thread. Only for OPs.
        flag, // Country code of poster
        poster_id; // Thread-level poster ID
    std::vector<Command> commands; // Results of hash commands
    std::map<uint64_t, LinkData> backlinks; // Posts linking to this post
    std::unordered_map<uint64_t, LinkData> links; // Posts linked by this post
    PostView* view = nullptr;

    Post() = default;

    // Parse from JSON
    Post(nlohmann::json&);

    ~Post() { delete view; }
};

// Contains thread metadata
class Thread {
public:
    uint64_t post_ctr, // Number of posts in thread
        image_ctr, // Number of images in thread
        reply_time, // Unix timestamp of last reply
        bump_time; // Unix timestamp of last bump
};
