#pragma once

#include "../../brunhild/view.hh"
#include "../posts/models.hh"
#include "../posts/view.hh"
#include "../state.hh"

// Render thread post count and expiration indicator
void render_post_counter();

// Render a thread page
void render_thread();

class ThreadView : public brunhild::ListView<Post, PostView> {
public:
    const unsigned long thread_id;

    ThreadView(unsigned long thread_id, std::string id = brunhild::new_id())
        : ListView("section", id)
        , thread_id(thread_id)
    {
        ThreadView::instances[thread_id] = this;
    }

    // All existing instaces
    static std::map<unsigned long, ThreadView*> instances;

    // TODO: Make this cleaner and without raw pointers
    static void clear();

protected:
    virtual std::vector<Post*> get_list();
    virtual std::shared_ptr<PostView> create_child(Post* p);
};
