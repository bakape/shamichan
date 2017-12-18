#include "../http.hh"
#include "../json.hh"
#include "../state.hh"
#include "../util.hh"
#include "connection.hh"
#include <functional>
#include <memory>
#include <unordered_map>

using nlohmann::json;
using std::shared_ptr;
using std::string;

// Contains data from the server for syncing open posts
struct OpenPost {
    bool has_image, spoilered;
    string body;
};

void send_sync_request()
{
    send_message(Message::synchronise,
        json({
            { "board", page->board },
            { "thread", page->thread },
        }));

    // TODO: Reclaim open posts
}

// Start listening to further updates fromserver
static void start_listening()
{

    // TODO: Hide loading indicator

    conn_SM->feed(ConnEvent::sync);
}

// Fetch post JSON and pass it to cb() on success
static void fetch_post(
    shared_ptr<int> jobs, unsigned long id, std::function<void(json)> cb)
{
    (*jobs)++;
    const string url = "/json/post/" + std::to_string(id);
    http_request(url, [=](unsigned short code, string data) {
        if (code != 200) {
            alert("Failed to fetch: " + url + " : " + std::to_string(code));
            return;
        }
        cb(json::parse(data));
        if (--(*jobs) == 0) {
            start_listening();
        }
    });
}

// Fetch a post that should be closed, but isn't
static void fetch_unclosed(shared_ptr<int> jobs, unsigned long id)
{
    fetch_post(jobs, id, [=](json data) {
        if (!posts->count(id)) { // Page navigated away from or something
            return;
        }
        auto& p = posts->at(id);
        p.extend(data);
        p.propagate_links();
        p.patch();
    });
}

// Sync open posts to the state they are in on the server's update feed
// dispatcher
static void sync_open_post(
    shared_ptr<int> jobs, unsigned long id, OpenPost data)
{
    // TODO
}

void synchronize(string data)
{
    // Skip posts before the first post in a shortened thread
    unsigned long min_id = 0;
    if (page->last_n) {
        min_id = -1;
        for (auto & [ _, p ] : *posts) {
            if (p.id < min_id && p.id != page->thread) {
                min_id = p.id;
            }
        }
        if (min_id == -1) { // No replies ;_;
            min_id = page->thread;
        }
    }

    shared_ptr<int> jobs = 0;
    auto j = json::parse(data);

    // Parse open posts into a map of structs
    std::unordered_map<unsigned long, OpenPost> open;
    auto& open_j = j["open"];
    open.reserve(open_j.size());
    for (json::iterator it = open_j.begin(); it != open_j.end(); it++) {
        auto& val = it.value();
        OpenPost p = { .body = val["body"] };
        if (val.count("hasImage")) {
            p.has_image = val["hasImage"];
        }
        if (val.count("spoilered")) {
            p.spoilered = val["spoilered"];
        }
        open[std::stoul(it.key())] = p;
    }

    for (auto & [ _, p ] : *posts) {
        if (p.editing && !open.count(p.id)) {
            fetch_unclosed(jobs, p.id);
        }
    }
    for (auto & [ id, data ] : open) {
        if (id >= min_id) {
            sync_open_post(jobs, id, data);
        }
    }

    // TODO: The rest of them

    if (*jobs == 0) { // Already synced
        start_listening();
    }
}
