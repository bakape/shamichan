#pragma once

#include "json.hh"
#include <string>
#include <tuple>
#include <type_traits>
#include <unordered_map>

// Contains the plugable langauge pack
class LanguagePack {
public:
    typedef std::unordered_map<std::string, std::string> StringMap;
    typedef std::unordered_map<std::string,
        std::tuple<std::string, std::string>>
        TupleMap;

    StringMap posts, // Definitions related to posts
        ui; // Related to UI

    TupleMap
        // Contains tuples of the word in singular and plural form
        plurals,
        // Data for rendering input forms
        forms;

    // Months names
    std::string calendar[12];

    // Day names
    std::string week[7];

    // Syncronization state labels
    std::string sync[5];

    // Load from inlined JSON in the DOM
    LanguagePack();

private:
    // Load <string, string> map from JSON
    void load_map(StringMap&, nlohmann::json&);

    // Load a map of string tuples from JSON
    void load_tuple_map(TupleMap&, nlohmann::json&);

    // Load an array of strings of known size from JSON
    template <class T> void load_array(T& arr, nlohmann::json& j);
};

// Contains the plugable langauge pack
extern LanguagePack const* lang;
