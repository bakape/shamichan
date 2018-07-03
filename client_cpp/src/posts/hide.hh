#pragma once

#include "../posts/models.hh"

// Remove all hidden posts from post collection.
// If recursive post hiding is enabled, recursively scan post list and remove
// all posts, that link hidden posts
void recurse_hidden_posts();

// Hide all posts that reply to post recursively, if enabled. Otherwise just
// hide this one post.
void hide_recursively(Post& post);
