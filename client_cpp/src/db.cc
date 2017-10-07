#include <emscripten.h>
#include <emscripten/bind.h>
#include <string>
#include <unordered_set>
#include <vector>

using emscripten::val;

const int db_version = 7;

// Database has errorred and all future calls should be ignored.
bool has_errored = false;

// Threads to load on the call from db_is_ready(). Keeps us from passing the
// thread ID array to JS, when opening the thread.
std::vector<uint64_t>* threads_to_load = nullptr;

// TODO: Deal with Firefox private Module
void load_db()
{
    EM_ASM_INT(
        {
            // Expiring post ID object stores
            window.postStores = ([
                // Posts created by this client
                'mine',
                // Posts hidden by client
                'hidden',
                // Replies to the user's posts that have already been seen
                'seen',
                // Posts that the user has viewed or scrolled past
                'seenPost',
            ]);

            window.handle_db_error = function(e)
            {
                Module.handle_db_error(e.toString());
            };

            var r = indexedDB.open('meguca', $0);
            r.onerror = function(e) { Module.handle_db_error(e.toString()); };
            r.onupgradeneeded = function(event)
            {
                var db = event.target.result;
                switch (event.oldVersion) {
                case 1:
                case 2:
                case 3:
                case 4:
                case 5:
                case 6:
                    // Delete all previous object stores
                    for (var i = 0; i < db.objectStoreNames; i++) {
                        db.deleteObjectStore(db.objectStoreNames[i]);
                    }

                    // Expiring post ID storage
                    for (var i = 0; i < postStores.length; i++) {
                        var s = db.createObjectStore(
                            postStores[i], { autoIncrement : true });
                    }
                    s.createIndex('expires', 'expires');
                    s.createIndex('op', 'op');

                    // Various miscellaneous objects
                    var main = db.createObjectStore('main', { keyPath : 'id' });
                    main.add({ id : 'background' });
                    main.add({ id : 'mascot' });
                }
            };
            r.onsuccess = function()
            {
                window.db = r.result;
                db.onerror = handle_db_error;

                // Reload this tab, if another tab requires a DB upgrade
                db.onversionchange = function()
                {
                    db.close();
                    location.reload(true);
                };

                Module.db_is_ready();

                // Delete expired keys from post ID object stores.
                // Delay for quicker starts.
                setTimeout(
                    function() {
                        for (var i = 0; i < postStores.length; i++) {
                            var name = postStores[i];
                            var t = db.transaction(name, 'readwrite');
                            t.onerror = handle_db_error;

                            var range = IDBKeyRange.upperBound(Date.now());
                            var req = t.objectStore(name)
                                          .index("expires")
                                          .openCursor(range);
                            req.onerror = handle_db_error;
                            req.onsuccess = function(event)
                            {
                                var cursor = event.result;
                                if (!cursor) {
                                    return;
                                }
                                cursor.delete();
                                cursor.continue();
                            };
                        }
                    },
                    10000);
            }
        },
        db_version);
}

void load_post_ids(const std::vector<uint64_t>& threads)
{
    EM_ASM_INT(
        {
            var ops = new Array($1);
            for (var i = 0; i < $1; i++) {
                ops[i] = getValue($0 + i * 8, 'i64')
            }

        },
        threads.data(), threads.size());
}

void handle_db_error(std::string err)
{
    has_errored = true;
    EM_ASM_INT({ console.error(Pointer_stringify($0)); }, err.c_str());
}

void db_is_ready()
{
    // TODO: Load actual thread IDS from post data
    threads_to_load = new std::vector<uint64_t>;
    threads_to_load->push_back(73);
    load_post_ids(*threads_to_load);
    delete threads_to_load;
}

EMSCRIPTEN_BINDINGS(module)
{
    emscripten::function("handle_db_error", &handle_db_error);
    emscripten::function("db_is_ready", &db_is_ready);
}
