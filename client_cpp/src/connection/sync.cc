#include "../state.hh"
#include "connection.hh"
#include <nlohmann/json.hpp>
#include <unordered_map>

using nlohmann::json;

const unsigned protocol_version = 1;

void send_sync_request()
{
    auto j = json({
        { "protocolVersion", protocol_version }, { "last100", page.last_100 },
        { "catalog", page.catalog }, { "board", page.board },
        { "page", page.page }, { "thread", page.thread },
    });
    send_message(Message::synchronise, j.dump());

    // TODO: Reclaim open posts
}
