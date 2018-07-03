#include "view.hh"
#include <algorithm>
#include <cctype>
#include <iterator>
#include <unordered_set>

using std::string;

enum token_type { unmatched, identifier, quoted, double_quoted, comment };

// Return, if char could be a part of an identifier in most languages
static bool is_identifier_char(const char b)
{
    return isalnum(b) || b == '_' || b == '$';
}

// Return, if word is one of the supported keywords
static bool is_keyword(const std::string& word)
{
    // Sorted for fast binary searching
    const static string keywords[] = { "NULL", "NaN", "abstract", "alias",
        "and", "arguments", "array", "asm", "assert", "async", "auto", "await",
        "base", "begin", "bool", "boolean", "break", "byte", "case", "catch",
        "char", "checked", "class", "clone", "compl", "const", "constexpr",
        "continue", "debugger", "decimal", "declare", "default", "defer",
        "deinit", "delegate", "delete", "do", "double", "echo", "elif", "else",
        "elseif", "elsif", "end", "ensure", "enum", "event", "except", "exec",
        "explicit", "export", "extends", "extension", "extern", "fallthrough",
        "false", "final", "finally", "fixed", "float", "fn", "for", "foreach",
        "friend", "from", "func", "function", "global", "go", "goto", "guard",
        "if", "impl", "implements", "implicit", "import", "in", "include",
        "inline", "inout", "instanceof", "int", "interface", "internal", "is",
        "lambda", "let", "lock", "long", "module", "mut", "mutable",
        "namespace", "native", "new", "next", "nil", "not", "null", "object",
        "operator", "or", "out", "override", "package", "params", "private",
        "protected", "protocol", "pub", "public", "raise", "readonly", "redo",
        "ref", "register", "repeat", "require", "rescue", "restrict", "retry",
        "return", "sbyte", "sealed", "short", "signed", "sizeof", "static",
        "str", "string", "struct", "subscript", "super", "switch",
        "synchronized", "template", "then", "throws", "transient", "true",
        "try", "type", "typealias", "typedef", "typeid", "typename", "typeof",
        "uint", "unchecked", "undef", "undefined", "union", "unless",
        "unsigned", "until", "use", "using", "var", "virtual", "void",
        "volatile", "when", "where", "while", "with", "xor", "yield" };

    return std::binary_search(std::begin(keywords), std::end(keywords), word);
}

// Return, if char is one of the supported operators
static bool is_operator(const char b)
{
    const static char operators[] = { '!', '%', '&', '*', '+', '-', '/', ':',
        '<', '=', '>', '?', '@', '^', '|', '~' };
    return std::binary_search(std::begin(operators), std::end(operators), b);
}

void PostView::highlight_syntax(std::string_view frag)
{
    if (!frag.size()) {
        return;
    }
    state.append({ "code", { { "class", "code-tag" } } }, true);
    state.buf.reserve(64);

    auto wrap_operator = [this](char op) {
        state.append(
            { "span", { { "class", "ms-operator" } }, string(1, op), true });
    };

    string token;
    token.reserve(64);
    token_type type = unmatched;
    char prev = 0;
    char b = 0;
    char next = 0;
    for (size_t i = 0; i < frag.size(); i++) {
        b = frag[i];
        next = i != frag.size() - 1 ? frag[i + 1] : 0;

        switch (type) {
        case unmatched:
            switch (b) {
            case '/':
                if (next == '/') {
                    type = comment;
                    state.append(
                        { "span", { { "class", "ms-comment" } } }, true);
                    state.buf += "//";
                    i++;
                } else {
                    wrap_operator('/');
                }
                break;
            case '\'':
                type = quoted;
                state.append({ "span", { { "class", "ms-string" } } }, true);
                state.buf += b;
                break;
            case '"':
                type = double_quoted;
                state.append({ "span", { { "class", "ms-string" } } }, true);
                state.buf += b;
                break;
            default:
                if (is_operator(b)) {
                    wrap_operator(b);
                } else if (is_identifier_char(b)) {
                    type = identifier;
                    token += b;
                } else {
                    state.buf += b;
                }
            }
            break;
        case identifier:
            token += b;
            if (!is_identifier_char(next)) {
                if (next == '(') {
                    state.append({ "span", { { "class", "ms-function" } },
                        token, true });
                } else if (is_keyword(token)) {
                    state.append({ "span", { { "class", "ms-operator" } },
                        token, true });
                } else {
                    state.buf += token;
                }
                type = unmatched;
                token.clear();
            }
            break;
        case quoted:
            state.buf += b;
            if (b == '\'' && prev != '\\') {
                type = unmatched;
                state.ascend();
            }
            break;
        case double_quoted:
            state.buf += b;
            if (b == '"' && prev != '\\') {
                type = unmatched;
                state.ascend();
            }
            break;
        case comment:
            state.buf += b;
            // We only have line-terminated commnets and those are terminated
            // upstream the call stack
            break;
        }

        prev = b;
    }

    // Flush any remaining buffer
    if (type == identifier) {
        state.buf += token;
    }

    // Close open tags
    if (type != unmatched) {
        state.ascend();
    }
    state.ascend();
}
