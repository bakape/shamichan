#pragma once

#include "util.hh"
#include <string>
#include <unordered_set>

// Open a connection to the IndexedDB database. Reports readiness to WaitGroup*.
void open_db(WaitGroup*);

// Load post ID sets from the database. Reports readiness to WaitGroup*.
void load_post_ids(WaitGroup*);
