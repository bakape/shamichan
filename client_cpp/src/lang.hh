#pragma once

#include "json.hh"
#include <string>
#include <tuple>
#include <type_traits>
#include <unordered_map>

// Contains the plugable langauge pack
class LanguagePack {
public:
    std::unordered_map<std::string, std::string>
        posts, // Definitions related to posts
        ui; // Related to UI

    // Contains tuples of the word in singular and plural form
    std::unordered_map<std::string, std::tuple<std::string, std::string>>
        plurals;

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
    void load_map(
        std::unordered_map<std::string, std::string>&, nlohmann::json&);

    // Load an array of strings of known size from JSON
    template <class T> void load_array(T& arr, nlohmann::json& j)
    {
        for (int i = 0; i < std::extent<T>::value; i++) {
            arr[i] = j[i];
        }
    }
};

// Contains the plugable langauge pack
extern LanguagePack const* lang;
