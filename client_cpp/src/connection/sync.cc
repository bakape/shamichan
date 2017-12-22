#include "../json.hh"
#include "../state.hh"
#include "connection.hh"
#include <unordered_map>

using nlohmann::json;

void send_sync_request()
{
    send_message(Message::synchronise,
        json({
                 { "newProtocol", true },
                 { "last100", page->last_n != 0 },
                 { "board", page->board },
                 { "thread", page->thread },
             })
            .dump());

    // TODO: Reclaim open posts
}
