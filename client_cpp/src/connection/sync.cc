#include "../json.hh"
#include "../state.hh"
#include "connection.hh"
#include <unordered_map>

using nlohmann::json;

void send_sync_request()
{
    auto j = json({
        { "newProtocol", true }, { "last100", page.last_100 },
        { "catalog", page.catalog }, { "board", page.board },
        { "page", page.page }, { "thread", page.thread },
    });
    send_message(Message::synchronise, j.dump());

    // TODO: Reclaim open posts
}
