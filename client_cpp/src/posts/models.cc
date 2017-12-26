#include "models.hh"
#include "../state.hh"
#include "hide.hh"
#include <sstream>

using json = nlohmann::json;
using std::string;

// Deserialize a property that might or might not be present from a kew of the
// same name
#define PARSE_OPT(key)                                                         \
    if (j.count(#key)) {                                                       \
        key = j[#key];                                                         \
    }

// Same as parse_opt, but explicitly converts to an std::string.
// Needed with std::optional<std::string> fields.
#define PARSE_OPT_STRING(key)                                                  \
    if (j.count(#key)) {                                                       \
        key = j.at(#key).get<string>();                                        \
    }

Image::Image(nlohmann::json& j)
{
    PARSE_OPT(apng);
    PARSE_OPT(audio);
    PARSE_OPT(video);
    PARSE_OPT(spoiler);
    file_type = static_cast<FileType>(j["fileType"]);
    thumb_type = static_cast<FileType>(j["thumbType"]);

    auto& j_dims = j["dims"];
    for (int i = 0; i < 4; i++) {
        dims[i] = j_dims[i];
    }

    PARSE_OPT(length);
    size = j["size"];
    PARSE_OPT_STRING(artist);
    PARSE_OPT_STRING(title);
    MD5 = j["MD5"];
    SHA1 = j["SHA1"];
    name = j["name"];
}

Command::Command(nlohmann::json& j)
{
    uint8_t _typ = j["type"];
    typ = static_cast<Type>(_typ);

    auto const& val = j["val"];
    switch (typ) {
    case Type::flip:
        flip = val;
        break;
    case Type::eight_ball:
        eight_ball = val;
        break;
    case Type::pyu:
        count = val;
        break;
    case Type::pcount:
        count = val;
        break;
    case Type::sync_watch:
        for (int i = 0; i < 5; i++) {
            sync_watch[i] = val[i];
        }
        break;
    case Type::dice:
        dice = val.get<std::vector<uint16_t>>();
        break;
    }
}

string Image::image_root() const
{
    if (config->image_root_override != "") {
        return config->image_root_override;
    }
    return "/assets/images";
}

string Image::thumb_path() const
{
    std::ostringstream s;
    s << image_root() << "/thumb/" << SHA1 << '.'
      << file_extentions.at(thumb_type);
    return s.str();
}

string Image::source_path() const
{
    std::ostringstream s;
    s << image_root() << "/src/" << SHA1 << '.'
      << file_extentions.at(file_type);
    return s.str();
}

void Post::extend(nlohmann::json& j)
{
    PARSE_OPT(editing);
    PARSE_OPT(deleted);
    PARSE_OPT(sage);
    PARSE_OPT(banned);
    PARSE_OPT(sticky);
    PARSE_OPT(locked);

    id = j["id"];
    PARSE_OPT(op);
    time = j["time"];

    body = j["body"];
    PARSE_OPT(board);
    PARSE_OPT_STRING(name);
    PARSE_OPT_STRING(trip);
    PARSE_OPT_STRING(auth);
    PARSE_OPT_STRING(subject);
    PARSE_OPT_STRING(flag);
    if (j.count("posterID")) {
        poster_id = j["posterID"].get<string>();
    }

    if (j.count("image")) {
        image = Image(j["image"]);
    }
    if (j.count("commands")) {
        auto& c = j["commands"];
        commands.clear(); // No to duplicate existing entries
        commands.reserve(c.size());
        for (auto& com : c) {
            commands.push_back(Command(com));
        }
    }
    if (j.count("links")) {
        auto& l = j["links"];
        links.reserve(l.size());
        for (auto& val : l) {
            links[val[0]] = { false, val[1] };
        }
    }
}

void Post::patch()
{
    // TODO: Check if post is not displayed? Not sure we will need this in the
    // future.

    // Proxy to top-most parent post, if inlined
    if (inlined_into) {
        return posts->at(inlined_into).patch();
    }

    VirtualView::patch(render());
}

void Post::propagate_links()
{

    // TODO: Notify about replies, if this post links to one of the user's posts

    for (auto && [ id, _ ] : links) {
        if (posts->count(id)) {
            auto& target = posts->at(id);
            target.backlinks[this->id] = LinkData{ false, this->op };
            target.patch();
        }
        if (post_ids->hidden.count(id)) {
            hide_recursively(*this);
        }
    }
}

void TextState::reset(Node* root)
{
    spoiler = false;
    quote = false;
    code = false;
    bold = false;
    italic = false;
    have_syncwatch = false;
    successive_newlines = 0;
    dice_index = 0;
    buf.clear();
    parents.clear();

    parents.push_back(root);
}

void TextState::append(Node n, bool descend, unsigned int gt_count)
{
    // Append escaped '>'
    for (unsigned int i = 0; i < gt_count; i++) {
        buf += "&gt;";
    }

    // Flush pending text node
    flush_text();

    parents.back()->children.push_back(n);
    if (descend) {
        parents.push_back(&parents.back()->children.back());
    }
}

void TextState::ascend()
{
    flush_text();
    parents.pop_back();
}

void TextState::flush_text()
{
    if (buf.size()) {
        Node n("span", buf, true);
        buf.clear();
        append(n);
    }
}
