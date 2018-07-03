#pragma once

#include "../../brunhild/view.hh"
#include "../posts/models.hh"
#include "../posts/view.hh"
#include "../state.hh"
#include "page.hh"

// Render thread post count and expiration indicator
void render_post_counter();

// Render a thread page
void render_thread();

class ThreadView : public brunhild::ListView<Post, PostView> {
public:
    const unsigned long thread_id;

    ThreadView(unsigned long thread_id, std::string id = brunhild::new_id());

    // All existing instaces
    static inline std::map<unsigned long, ThreadView*> instances;
    static void clear() { ThreadView::instances.clear(); }

protected:
    virtual std::vector<Post*> get_list();
    std::shared_ptr<PostView> create_child(Post* p);
};

// Contains the post-related portion of the thread page
class ThreadPageView : public PageView {
protected:
    brunhild::View* thread_container() { return new ThreadView(page.thread); }
    std::vector<brunhild::View*> top_controls();
    std::vector<brunhild::View*> bottom_controls();
};
