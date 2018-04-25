// Hash command parsing and rendering

#include "../lang.hh"
#include "../state.hh"
#include "models.hh"
#include <cctype>
#include <ctime>
#include <iomanip>
#include <sstream>
#include <unordered_map>
#include <utility>

using std::nullopt;
using std::optional;
using std::ostringstream;
using std::string;
using std::string_view;
using std::unordered_map;

long server_time_offset = 0;

// IDs of posts, that are pending a rerender to update the syncwatch and the
// time they should be rerender at. Specifying a timestamp helps avoid useless
// subtree diffs.
static unordered_map<unsigned long, time_t> pending_rerender;

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
            if (posts.count(id)) {
                posts.at(id).patch();
            }
        }
    }
}

// Read any digit from string_view and return it, if any.
// Rejects numbers longer than 5 digits.
static optional<unsigned> parse_uint(string_view& word)
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

// If num is made of the same digit repeating
static bool check_em(unsigned num)
{
    if (num < 10) {
        return false;
    }
    const auto digit = num % 10;
    while (1) {
        num /= 10;
        if (!num) {
            return true;
        }
        if (num % 10 != digit) {
            return false;
        }
    }
}

// Parse dice rolls and return inner command string and formatting class, if
// matched
static std::pair<string, string> parse_dice(
    string& name, string_view word, const Command& val)
{
    unsigned dice = 1;
    unsigned faces = 0;

    // Has leading digits
    if (name == "") {
        if (auto d = parse_uint(word)) {
            dice = *d;
        } else {
            return {};
        }
        if (!word.size() || word[0] != 'd') {
            return {};
        }
        word = word.substr(1);
    }

    if (auto f = parse_uint(word)) { // Must consume the rest of the text
        faces = *f;
    } else {
        return {};
    }
    if (word.size() || dice > 10 || faces > 10000) {
        return {};
    }

    // Rebuild command syntax
    name.clear();
    if (dice != 1) {
        name += std::to_string(dice);
    }
    name += 'd' + std::to_string(faces);

    ostringstream os;
    unsigned sum = 0;
    for (auto roll : std::get<std::array<uint16_t, 10>>(val.val)) {
        if (!roll) { // Array is zero padded
            break;
        }
        if (sum) {
            os << " + ";
        }
        sum += roll;
        os << roll;
    }
    if (dice > 1) {
        os << " = " << sum;
    }

    // Determine roll formatting class
    string cls;
    const unsigned max_roll = dice * faces;
    if (max_roll >= 10 && faces != 1) { // no special formatting for small rolls
        if (max_roll == sum) {
            cls = "super_roll";
        } else if (sum == dice) {
            cls = "kuso_roll";
        } else if (sum == 69 || sum == 6969) {
            cls = "lewd_roll";
        } else if (check_em(sum)) {
            if (sum < 100) {
                cls = "dubs_roll";
            } else if (sum < 1000) {
                cls = "trips_roll";
            } else if (sum < 10000) {
                cls = "quads_roll";
            } else {
                cls = "rainbow_roll";
            }
        }
    }

    return { os.str(), cls };
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
    string cls;
    auto const& val = commands[state.dice_index];
    if (name == "flip") {
        check_consumed;
        inner = std::get<bool>(val.val) ? "flap" : "flop";
    } else if (name == "8ball") {
        check_consumed;
        inner = val.eight_ball;
    } else if (name == "rcount") {
        check_consumed;
        inner = std::to_string(std::get<unsigned long>(val.val));
    } else if (name == "roulette") {
        check_consumed;
        const auto arr = std::get<std::array<uint8_t, 2>>(val.val);
        inner = std::to_string(arr[0]) + '/' + std::to_string(arr[1]);
        if (arr[0] == 1) {
            cls = "dead";
        }
    } else if (name == "sw") {
        return parse_syncwatch(word);
    } else {
        auto p = parse_dice(name, word, val);
        inner = p.first;
        cls = p.second;
    }
    if (inner == "") {
        return nullopt;
    }

    state.dice_index++;
    ostringstream os;
    os << '#' << name << " (" << inner << ')';

    return { { "strong", { { "class", cls } }, os.str(), true } };
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
        = std::get<std::array<unsigned long, 5>>(
            commands[state.dice_index++].val);
    const unsigned long now = std::time(0) + server_time_offset;
    ostringstream s;
    if (now > end) {
        s << lang.ui.at("finished");
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
            "em", {},
            {
                {
                    "strong", { { "class", "embed syncwatch" } }, s.str(),
                },
            },
        },
    };
}
