#include "posts.hh"
#include "../../brunhild/mutations.hh"
#include "../page/thread.hh"
#include "../posts/models.hh"
#include "../state.hh"
#include <nlohmann/json.hpp>

using nlohmann::json;

void insert_post(std::string_view msg)
{
    // TODO: R/a/dio song name override

    auto j = json::parse(msg);
    Post p(j);

    // TODO: Existing post (created by this client) handling

    p.op = page.thread;
    p.board = page.board;

    // Need to ensure the post is queued to render and in the global collection
    // for all further operations
    (posts)[p.id] = p;
    auto& ref = posts.at(p.id);
    if (!ref.editing) {
        ref.propagate_links();
    }

    auto& t = threads.at(page.thread);
    t.post_ctr++;
    if (ref.image) {
        t.image_ctr++;
    }
    ThreadView::instances[page.thread]->patch();
    render_post_counter();

    // TODO: Unread post counting
}
