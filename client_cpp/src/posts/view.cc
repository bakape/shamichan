#include "view.hh"
#include "../state.hh"

Post* PostView::get_model()
{
    return posts.count(model_id) ? &posts.at(model_id) : NULL;
}

void PostView::patch()
{
    // TODO: Check if post is not displayed? Not sure we will need this in the
    // future.

    // Proxy to top-most parent post, if inlined
    if (const auto inlined_into = get_model()->inlined_into; inlined_into) {
        return posts.at(inlined_into).patch();
    }

    ModelView::patch();
}

void PostView::remove()
{
    View::remove();
    this->~PostView();
}

void TextState::reset(Node* root)
{
    spoiler = false;
    quote = false;
    code = false;
    bold = false;
    italic = false;
    red = false;
    blue = false;
    have_syncwatch = false;
    successive_newlines = 0;
    dice_index = 0;
    buf.clear();
    parents.clear();

    parents.push_back(root);
}

void TextState::append(Node n, bool descend, unsigned gt_count)
{
    // Append escaped '>'
    for (unsigned i = 0; i < gt_count; i++) {
        buf += "&gt;";
    }

    // Flush pending text node
    flush_text();

    parents.back()->children.push_back(n);
    if (descend) {
        parents.push_back(&parents.back()->children.back());
    }
}

void TextState::ascend()
{
    flush_text();
    parents.pop_back();
}

void TextState::flush_text()
{
    if (buf.size()) {
        Node n("span", buf, true);
        buf.clear();
        append(n);
    }
}
