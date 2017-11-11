// Hash command parsing and rendering

#include "models.hh"
#include <cctype>
#include <sstream>

using std::nullopt;
using std::optional;
using std::ostringstream;
using std::string;
using std::string_view;

// Read any digit from string_view and return it. Returns 0 on no match.
static unsigned int read_uint(string_view& word)
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
    if (num.size() > 5) {
        return 0;
    }
    if (num.size()) {
        return std::stoul(num);
    }
    return 0;
}

// Parse dice rolls and return inner command string, if matched
static string parse_dice(string& name, string_view word, const Command& val)
{
    unsigned int dice = 0;
    unsigned int faces = 0;

    // Has leading digits
    if (name == "") {
        dice = read_uint(word);
        if (!word.size() || word[0] != 'd') {
            return "";
        }
        word = word.substr(1);
    }
    name = string(word);

    faces = read_uint(word); // Should consume the rest of the text
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

// TODO
// TODO: Also need to figure out, how to handle updating these on countdown.
// Perhaps a global registry, that gets flushed on page re-render?
optional<Node> Post::parse_syncwatch(std::string_view frag) { return nullopt; }
