#pragma once

#include "thread.hh"

// Render a board or catalog page
void render_board();

// TODO: Deleted thread toggle
class BoardThreadView : public ThreadView {
    using ThreadView::ThreadView;

protected:
    brunhild::Attrs attrs() const { return { { "class", "index-thread" } }; }
};
