#include "models.hh"
#include "../state.hh"
#include <sstream>

using json = nlohmann::json;
using std::string;

// Deserialize a property that might or might not be present from a kew of the
// same name
#define parse_opt(key)                                                         \
    if (j.count(#key)) {                                                       \
        key = j[#key];                                                         \
    }

// Same as parse_opt, but explicitly converts to an std::string.
// Needed with std::optional<std::string> fields.
#define parse_opt_string(key)                                                  \
    if (j.count(#key)) {                                                       \
        key = j.at(#key).get<string>();                                        \
    }

Image::Image(nlohmann::json& j)
{
    parse_opt(apng);
    parse_opt(audio);
    parse_opt(video);
    parse_opt(spoiler);
    file_type = static_cast<FileType>(j["fileType"]);
    thumb_type = static_cast<FileType>(j["thumbType"]);

    auto& j_dims = j["dims"];
    for (int i = 0; i < 4; i++) {
        dims[i] = j_dims[i];
    }

    parse_opt(length);
    size = j["size"];
    parse_opt_string(artist);
    parse_opt_string(title);
    MD5 = j["MD5"];
    SHA1 = j["SHA1"];
    name = j["name"];
}

Command::Command(nlohmann::json& j)
{
    typ = static_cast<Type>(j["type"]);

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

Post::Post(nlohmann::json& j)
{
    parse_opt(editing);
    parse_opt(deleted);
    parse_opt(sage);
    parse_opt(banned);
    parse_opt(sticky);
    parse_opt(locked);

    id = j["id"];
    parse_opt(op);
    time = j["time"];

    body = j["body"];
    parse_opt(board);
    parse_opt_string(name);
    parse_opt_string(trip);
    parse_opt_string(auth);
    parse_opt_string(subject);
    parse_opt_string(flag);
    if (j.count("posterID")) {
        poster_id = j["posterID"].get<string>();
    }

    if (j.count("image")) {
        image = Image(j["image"]);
    }
    if (j.count("commands")) {
        auto& c = j["commands"];
        commands.reserve(c.size());
        for (auto& com : c) {
            commands.push_back(Command(com));
        }
    }
    if (j.count("links")) {
        auto& l = j["links"];
        links.reserve(l.size());
        for (auto& val : l) {
            links[val[0]] = {.op = val[1] };
        }
    }
}
