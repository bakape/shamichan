#include "../options/options.hh"
#include "../state.hh"
#include "models.hh"
#include <unordered_set>

// Recursively gather links of posts linking to the parent post
static void recurse_backlinks(
    const std::map<unsigned long, LinkData>& backlinks,
    std::unordered_set<unsigned long>& to_hide)
{
    for (auto & [ id, _ ] : backlinks) {
        // Skip posts already marked as hidden. Those will have or have already
        // had posts linking them marked recursively by other calls
        if (!post_ids.hidden.count(id) && posts.count(id)) {
            post_ids.hidden.insert(id);
            to_hide.insert(id);
            recurse_backlinks(posts.at(id).backlinks, to_hide);
        }
    }
}

void recurse_hidden_posts()
{
    std::unordered_set<unsigned long> to_hide;

    const bool recurse = options.hide_recursively;
    for (auto const & [ id, p ] : posts) {
        if (post_ids.hidden.count(id)) {
            to_hide.insert(id);
            if (recurse) {
                recurse_backlinks(p.backlinks, to_hide);
            }
        }
    }

    for (auto id : to_hide) {
        posts.erase(id);
    }
}

void hide_recursively(Post& post)
{
    std::unordered_set<unsigned long> to_hide;
    post_ids.hidden.insert(post.id);
    if (options.hide_recursively) {
        recurse_backlinks(post.backlinks, to_hide);
    }

    post.remove();
    for (auto id : to_hide) {
        if (posts.count(id)) {
            posts.at(id).remove();
        }
    }
}
