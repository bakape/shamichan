#include "../options/options.hh"
#include "../state.hh"
#include "models.hh"

// Set posts linking this post as hidden, but do not persist this
static void set_linking_hidden(
    const std::map<unsigned long, LinkData>& backlinks)
{
    for (auto & [ id, _ ] : backlinks) {
        // Skip posts already marked as hidden. Those will have or have already
        // had posts linking them marked recursively by other calls
        if (!post_ids->hidden.count(id) && posts->count(id)) {
            post_ids->hidden.insert(id);
            set_linking_hidden(posts->at(id).backlinks);
        }
    }
}

void recurse_hidden_posts()
{
    if (!options->hide_recursively) {
        return;
    }

    for (auto const & [ _, p ] : *posts) {
        if (post_ids->hidden.count(p.id)) {
            set_linking_hidden(p.backlinks);
        }
    }
}
