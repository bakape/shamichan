#pragma once

#include <array>
#include <functional>
#include <map>
#include <nlohmann/json.hpp>
#include <optional>
#include <stdint.h>
#include <string>
#include <string_view>
#include <tuple>
#include <unordered_map>
#include <variant>
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
const static std::unordered_map<FileType, std::string> file_extentions = {
    { FileType::jpg, "jpg" }, { FileType::png, "png" },
    { FileType::gif, "gif" }, { FileType::webm, "webm" },
    { FileType::pdf, "pdf" }, { FileType::svg, "svg" },
    { FileType::mp4, "mp4" }, { FileType::mp3, "mp3" },
    { FileType::ogg, "ogg" }, { FileType::zip, "zip" }, { FileType::_7z, "7z" },
    { FileType::targz, "tar.gz" }, { FileType::tarxz, "tar.xz" },
    { FileType::flac, "flac" }, { FileType::txt, "txt" },
};

struct Image {
    bool apng = false, // PNG image is APNG
        audio = false, // Has audio
        video = false, // Has video
        spoiler = false; // Is spoilered
    FileType file_type, // File type of source file
        thumb_type; // File type of thumbnail
    uint16_t dims[4];
    uint32_t length = 0; // Length of media, if a media file
    unsigned long size;
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
    enum class Type : uint8_t {
        dice,
        flip,
        eight_ball,
        sync_watch,
        pyu,
        pcount,
        roulette,
        rcount
    } typ;

    // Use typ, to get out the relevant value
    std::variant<bool, unsigned long, std::array<unsigned long, 5>,
        std::array<uint16_t, 10>, std::array<uint8_t, 2>>
        val;
    std::string eight_ball; // Result of #8ball command

    // Parse from JSON
    Command(nlohmann::json&);
};

// Data associated with link to another post. Is always pared in a map with
// the ID of the linked post as a key.
struct LinkData {
    // The post and its subtree is now a child of the link
    // TODO: This should be in a global collection
    bool is_inlined = false;
    // Parent thread ID of the post
    unsigned long op;
    // Parent board id
    std::string board;
};

class PostView;

// Generic post model
struct Post {
    // Post is currrently being edited
    bool editing = false,
         // Deleted by moderator
        deleted = false,
         // Poster disabled bumping of the parent thread
        sage = false,
         // Banned for this post by moderator
        banned = false,
         // Thread is stickied. Only for OPs.
        sticky = false,
         // Thread is locked. Only for OPs.
        locked = false,
         // The user has already seen this post
        seen = false;

    std::optional<Image> image;

    // ID of post this post is currently inlined into, if any
    // TODO: This should be in a global collection
    unsigned long inlined_into = 0;

    unsigned long id, op;

    time_t time;

    std::string body, board;

    std::optional<std::string> name, // Name of poster
        trip, // Trip code of poster
        auth, // Staff title of poster
        flag, // Country code of poster
        poster_id; // Thread-level poster ID

    std::vector<Command> commands; // Results of hash commands

    // Posts linking to this post. Backlinks need to be sorted, thus std::map.
    std::map<unsigned long, LinkData> backlinks;

    std::unordered_map<unsigned long, LinkData>
        links; // Posts linked by this post

    // Views associated to this post. There can be multiple, because of various
    // previews and such.
    std::vector<std::shared_ptr<PostView>> views;

    Post() = default;

    // Parse from JSON
    Post(nlohmann::json& j) { extend(j); }

    // Extend post data by parsing new values from JSON
    void extend(nlohmann::json&);

    // Patches all views associated with this post
    void patch();

    // Check if this post replied to one of the user's posts and trigger
    // handlers.
    // Set and render backlinks on any linked posts.
    void propagate_links();

    // Parse link data from JSON
    void parse_links(nlohmann::json&);

    // Parse hash command results from JSON
    void parse_commands(nlohmann::json&);

    // Close a post being edited
    void close();
};

#include "view.hh"

// Contains thread metadata
struct Thread {
    bool deleted = false, // Thread deleted by staff
        locked = false, // Thread locked by staff
        sticky = false, // Stuck to board page top by stuff
        non_live = false; // Live post updates disabled in thread
    unsigned long id, // ID of the thread
        time, // Creation time
        post_ctr, // Number of posts in thread
        image_ctr, // Number of images in thread
        reply_time, // Unix timestamp of last reply
        bump_time; // Unix timestamp of last bump
    std::string board, // Parent board
        subject; // Thread subject
};
