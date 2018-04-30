#pragma once

#include "../../brunhild/view.hh"
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

using brunhild::Node;

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

class Image {
public:
    bool apng = false, // PNG image is APNG
        audio = false, // Has audio
        video = false, // Has video
        spoiler = false, // Is spoilered
        expanded = false, // Expand image thumbnail to full view
        reveal_thumbnail = false; // Reveal a hidden image with [Show]
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
        _,
        __,
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
    bool is_inlined = false;
    // Parent thread ID of the post
    unsigned long op;
    // Parent board id
    std::string board;
};

// State of a post's text. Used for adding enclosing tags to the HTML while
// parsing.
class TextState {
public:
    bool spoiler = false, // Current text is spoilered
        quote = false, // Current line is spoilered
        code = false, // Text is inside code block
        bold = false, // Text inside bold tag
        italic = false, // Text inside italic tag
        red = false, // Text inside red color tag
        blue = false, // Text inside blue color tag
        have_syncwatch = false; // Text contains #syncwatch command(s)
    int successive_newlines = 0; // Number of successive newlines in text
    size_t dice_index = 0; // Index of the next dice array item to use

    // Used for building text nodes. Flushed on append() or ascend().
    std::string buf;

    // Reset to initial values and sets Node as the new root parent.
    void reset(Node* root);

    // Append a Node to the current lowermost parent.
    // If descend = true, make it the next parent to append to.
    // gt_count prepends "greater than" symbols in the text node before this
    // node.
    void append(Node n, bool descend = false, unsigned gt_count = 0);

    // Acsend one level up the parent tree and make it the next node to append
    // to
    void ascend();

    // Flush text buffer into escaped text node, if not empty
    void flush_text();

    // Layers of formatting for the parser to descend.
    // <em> not included, as it is line-based.
    static const int tag_depth = 5;

    // Returns all flags as array ordered by parent to child
    inline std::array<bool, tag_depth> as_array() const
    {
        return { { spoiler, bold, italic, red, blue } };
    }

private:
    // Last child nodes of the blockquote subtree.
    // Used to keep track of nodes to append to, while populating the
    // subtree.
    std::vector<Node*> parents;
};

// Generic post model
class Post : public brunhild::View {
public:
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

    Post() = default;

    // Parse from JSON
    Post(nlohmann::json& j) { extend(j); }

    // Extend post data by parsing new values from JSON
    void extend(nlohmann::json&);

    // Generates the model's node tree
    Node render();

    // Patch the current contents of the post into the DOM.
    // If the post is currently inlined into another post, this method will
    // delegate the patch to the topmost parent.
    void patch();

    // Remove a post from global collection and it's associated DOM element,
    // if any
    void remove();

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

private:
    TextState state;

    // Render the header on top of the post
    Node render_header();

    // Render the name and tripcode in the header
    Node render_name();

    // Renders a time element. Can be either absolute or relative.
    Node render_time();

    // Render the information caption above the image.
    Node render_figcaption();

    // Render reverse image search links
    Node render_image_search();

    // Render uploaded file meta information
    Node render_file_info();

    // Render a thumbnail or expanded source media content
    std::tuple<Node, std::optional<Node>> render_image();

    // Render the text body of a post
    Node render_body();

    // Parse temporary links in open posts, that still may be edited
    void parse_temp_links(std::string_view);

    // Parse a line fragment into an HTML subtree
    void parse_fragment(std::string_view);

    // Highlight common programming code syntax
    void highlight_syntax(std::string_view);

    // Function run on a separated string fragment
    typedef std::function<void(std::string_view)> OnFrag;

    // Detect and format code tags. Call fn on unmatched sub-fragments.
    void parse_code(std::string_view frag, OnFrag fn);

    // Inject spoiler tags and call fn on the remaining parts
    void parse_spoilers(std::string_view frag, OnFrag fn);

    // Open and close any tags up to level, if they are set.
    // Increment level by 1 for each tag deeper you go.
    void wrap_tags(int level);

    // Inject bold tags and call fn on the remaining parts
    void parse_bolds(std::string_view frag, OnFrag fn);

    // Inject italic tags and call fn on the remaining parts
    void parse_italics(std::string_view frag, OnFrag fn);

    // Inject red color tags and call fn on the remaining parts
    void parse_reds(std::string_view frag, OnFrag fn);

    // Inject red color tags and call fn on the remaining parts
    void parse_blues(std::string_view frag, OnFrag fn);

    // Parse a string into words and call fn on each word.
    // Handles space padding and leading/trailing punctuation.
    void parse_words(std::string_view frag, OnFrag fn);

    // Parse internally-defined or board reference URL.
    // Returns preceding '>' count and link Node, if matched.
    std::optional<std::tuple<int, Node>> parse_reference(std::string_view word);

    // Parse hash commands. Return Node, if matched.
    std::optional<Node> parse_commands(std::string_view word);

    // Parse syncwatch command and return Node, if matched
    std::optional<Node> parse_syncwatch(std::string_view frag);
};

// Contains thread metadata
class Thread {
public:
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
