// Hash command parsing and rendering

#include "../lang.hh"
#include "../state.hh"
#include "models.hh"
#include <cctype>
#include <ctime>
#include <iomanip>
#include <sstream>
#include <unordered_map>

using std::nullopt;
using std::optional;
using std::ostringstream;
using std::string;
using std::string_view;
using std::unordered_map;

// IDs of posts, that are pending a rerender to update the syncwatch and the
// time they should be rerender at. Specifying a timestamp helps avoid useless
// subtree diffs.
unordered_map<unsigned long, time_t> pending_rerender;

void rerender_syncwatches()
{
    if (!pending_rerender.size()) {
        return;
    }

    // Prevent modifications from patching affecting the iterated map
    unordered_map<unsigned long, time_t> m(pending_rerender);
    const auto now = std::time(0);

    for (auto[id, when] : m) {
        if (now >= when) {
            pending_rerender.erase(id);

            // Posts might have been removed by now
            if (posts->count(id)) {
                posts->at(id).patch();
            }
        }
    }
}

// Read any digit from string_view and return it, if any.
// Rejects numbers longer than 5 digits.
static optional<unsigned int> parse_uint(string_view& word)
{
    string num;
    num.reserve(5);

    while (word.size()) {
        if (isdigit(word[0])) {
            num += word[0];
            word = word.substr(1);
        } else {
            break;
        }
    }
    if (num.size() > 5 || !num.size()) {
        return nullopt;
    }
    return { std::stoul(num) };
}

// Parse dice rolls and return inner command string, if matched
static string parse_dice(string& name, string_view word, const Command& val)
{
    unsigned int dice = 0;
    unsigned int faces = 0;

    // Has leading digits
    if (name == "") {
        if (auto d = parse_uint(word)) {
            dice = *d;
        } else {
            return "";
        }
        if (!word.size() || word[0] != 'd') {
            return "";
        }
        word = word.substr(1);
    }

    // Rebuild command syntax
    name.clear();
    if (dice) {
        name += std::to_string(dice);
    }
    name += 'd' + std::to_string(faces);

    if (auto f = parse_uint(word)) { // Must consume the rest of the text
        faces = *f;
    } else {
        return "";
    }
    if (word.size() || dice > 10 || faces > 10000) {
        return "";
    }

    ostringstream os;
    unsigned int sum = 0;
    for (auto roll : val.dice) {
        if (sum) {
            os << " + ";
        }
        sum += roll;
        os << roll;
    }
    if (val.dice.size() > 1) {
        os << " = " << sum;
    }
    return os.str();
}

optional<Node> Post::parse_commands(string_view word)
{
    // Guard against invalid dice rolls
    if (state.dice_index >= commands.size()) {
        return nullopt;
    }

    // Strip leading hash
    word = word.substr(1);

    // Attempt to read command name
    string name;
    name.reserve(word.size());
    while (word.size()) {
        const char ch = word[0];
        if (islower(ch) || ch == '8') {
            name += ch;
            word = word.substr(1);
        } else {
            break;
        }
    }

// Did not consume entire expression and no arguments possible
// -> it's invalid
#define check_consumed                                                         \
    if (word.size()) {                                                         \
        return nullopt;                                                        \
    }

    string inner;
    auto const& val = commands[state.dice_index];
    if (name == "flip") {
        check_consumed;
        inner = val.flip ? "flap" : "flop";
    } else if (name == "8ball") {
        check_consumed;
        inner = val.eight_ball;
    } else if (name == "pyu" || name == "pcount") {
        check_consumed;
        inner = std::to_string(val.count);
    } else if (name == "sw") {
        return parse_syncwatch(word);
    } else {
        inner = parse_dice(name, word, val);
    }
    if (inner == "") {
        return nullopt;
    }

    state.dice_index++;
    ostringstream os;
    os << '#' << name << " (" << inner << ')';

    return { { "strong", os.str(), true } };
}

// TODO: Also need to figure out, how to handle updating these on countdown.
// Perhaps a global registry, that gets flushed on page re-render?
// Probably a good idea to hook these before RAF execution.
optional<Node> Post::parse_syncwatch(std::string_view frag)
{
    using std::setw;

    // Parse and validate
    if (!frag.size()) {
        return nullopt;
    }
    if (!parse_uint(frag) || frag.size() < 2 || frag[0] != ':') {
        return nullopt;
    }
    frag = frag.substr(1);
    if (!parse_uint(frag)) {
        return nullopt;
    }

    // Hour parameter is optional, so only the first 2 parameters are required
    if (frag.size() && frag[0] == ':') {
        frag = frag.substr(1);
        if (!parse_uint(frag)) {
            return nullopt;
        }
    }

    // Validate optional offset parameter
    if (frag.size()) {
        if (frag.size() < 2) {
            return nullopt;
        }
        switch (frag[0]) {
        case '+':
        case '-':
            frag = frag.substr(1);
            break;
        default:
            return nullopt;
        }

        // Must be fully consumbed now
        if (!parse_uint(frag) || frag.size()) {
            return nullopt;
        }
    }

    // Format inner string
    // TODO: Apply offset from server clock
    const auto[hours, min, sec, start, end]
        = commands[state.dice_index++].sync_watch;
    const unsigned long now = std::time(0);
    ostringstream s;
    if (now > end) {
        s << lang->ui.at("finished");
    } else {
        if (now < start) {
            s << start - now;
        } else {
            unsigned long diff = now - start;
            const auto hours_elapsed = diff / 3600;
            diff %= 3600;
            const auto min_elapsed = diff / 60;
            diff %= 60;

            s << std::setfill('0') << setw(2) << hours_elapsed << ':' << setw(2)
              << min_elapsed << ':' << setw(2) << diff << " / " << setw(2)
              << hours << ':' << setw(2) << min << ':' << setw(2) << sec;
        }

        // Schedule next render to update counter
        pending_rerender[id] = now + 1;
    }

    return {
        {
            "em",
            {},
            {
                {
                    "strong",
                    { { "class", "embed syncwatch" } },
                    s.str(),
                },
            },
        },
    };
}
