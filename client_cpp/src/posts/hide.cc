#include "../options/options.hh"
#include "../state.hh"
#include "models.hh"

// Set posts linking this post as hidden, but do not persist this.
// If patch = true, patch changes on hidden posts.
static void set_linking_hidden(
    const std::map<unsigned long, LinkData>& backlinks, bool patch = false)
{
    for (auto & [ id, _ ] : backlinks) {
        // Skip posts already marked as hidden. Those will have or have already
        // had posts linking them marked recursively by other calls
        if (!post_ids->hidden.count(id) && posts->count(id)) {
            post_ids->hidden.insert(id);
            auto& target = posts->at(id);
            if (patch) {
                target.patch();
            }
            set_linking_hidden(target.backlinks);
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

void hide_recursively(Post& post)
{
    if (post_ids->hidden.count(post.id)) { // Already hidden
        return;
    }

    post_ids->hidden.insert(post.id);
    post.patch();
    if (options->hide_recursively) {
        set_linking_hidden(post.backlinks, true);
    }
}
