#pragma once

#include <string>

// Send a requests to the server to synchronise to the current page and
// subscribe to the appropriate event feeds
void send_sync_request();

// Synchronise to the server and start receiving updates on the appropriate
// channel. If there are any missed messages, fetch them.
void synchronize(std::string data);
