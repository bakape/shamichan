#pragma once

#include "../../brunhild/view.hh"
#include "models.hh"
#include <memory>

using brunhild::Node;

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

class PostView : public brunhild::ModelView<Post> {
    using brunhild::ModelView<Post>::render;

public:
    // ID of model the post is associated to
    const unsigned long model_id;

    bool expanded = false, // Expand image thumbnail to full view
        reveal_thumbnail = false; // Reveal a hidden image with [Show]

    // id: parent model id
    PostView(unsigned long model_id)
        : model_id(model_id)
    {
    }

    // Patch the current contents of the post into the DOM.
    // If the post is currently inlined into another post, this method will
    // delegate the patch to the topmost parent.
    void patch();

    Post* get_model();

private:
    TextState state;

    // Posts inlined into this post's links
    std::unordered_map<unsigned long, std::unique_ptr<PostView>> inlined_posts;

    // Generates the model's node tree
    Node render(Post*);

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

    // Renders link to other posts and any inlined posts inside
    Node render_link(unsigned long id, const LinkData& data);

    // Parse hash commands. Return Node, if matched.
    std::optional<Node> parse_commands(std::string_view word);

    // Parse syncwatch command and return Node, if matched
    std::optional<Node> parse_syncwatch(std::string_view frag);
};
