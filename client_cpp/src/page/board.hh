#pragma once

#include "thread.hh"

// Render a board or catalog page
void render_board();

// TODO: Deleted thread toggle
class BoardThreadView : public ThreadView {
    using ThreadView::ThreadView;

protected:
    brunhild::Attrs attrs() { return { { "class", "index-thread" } }; }
};

// Contains the post-related portion of the board page
class BoardPageView : public PageView {
protected:
    brunhild::View* thread_container()
    {
        return new brunhild::NodeView({ "span", "TODO" });
    }
};
