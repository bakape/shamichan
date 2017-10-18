#pragma once

#include "../../brunhild/view.hh"
#include "models.hh"
#include <ctime>
#include <memory>
#include <optional>
#include <string>

using brunhild::Node;

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

// Base post view class
class PostView : public brunhild::VirtualView {
public:
    // Initializes a new PostView with data from model
    PostView(const Post& model)
        : brunhild::VirtualView(render(model))
    {
    }

private:
    // Generates the post's node tree
    Node render(const Post&);

    // Render the header on top of the post
    Node render_header(const Post&);

    // Render the name and tripcode in the header
    Node render_name(const Post&);

    // Renders a time element. Can be either absolute or relative.
    Node render_time(time_t);

    // Render the information caption above the image.
    // Set reveal to true, if in hidden thumbnail mode, to reveal the thumbnail.
    Node render_figcaption(const Image& img, bool reveal = false);

    // Render reverse image search links
    Node render_image_search(const Image& img);

    // Render uploaded file meta information
    Node render_file_info(const Image& img);

    bool expanded = false, // Thumbnail is expanded
        taller_than_viewport = false, // Image is taller than the viewport
        revealed = false; // Revealing a hidden image with [Show]

    TextState state;
};

// Renders readable elapsed time since Unix timestamp then
std::string relative_time(time_t then);
