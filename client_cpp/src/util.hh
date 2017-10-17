#pragma once

#include <string>
#include <tuple>

// Read inner HTML from DOM element by ID
std::string get_inner_html(const std::string& id);

// Return either the singular or plural form of a translation, depending on n
std::string pluralize(int n, const std::tuple<std::string, std::string>& word);

// Write n to out padded to 2 digits
void pad(std::string& out, unsigned int n);

// Cast a C string represented as int to std::string and free the original
std::string convert_c_string(int);

// URL encode s and return the new encoded string
std::string url_encode(const std::string& s);

// Log string to JS console
void console_log(const std::string&);
