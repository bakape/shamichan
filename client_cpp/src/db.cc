#include "state.hh"
#include <emscripten.h>
#include <emscripten/bind.h>
#include <string>
#include <unordered_set>
#include <vector>

const int db_version = 7;

// Database has errorred and all future calls should be ignored.
bool has_errored = false;

// Threads to load on the call from db_is_ready(). Keeps us from passing the
// thread ID array to JS, when opening the thread.
std::unordered_set<uint64_t>* threads_to_load = nullptr;

// TODO: Deal with Firefox private Module
void load_db(std::unordered_set<uint64_t> thread_ids)
{
    threads_to_load = new std::unordered_set<uint64_t>(thread_ids);

    EM_ASM_INT(
        {
            // Expiring post ID object stores
            window.postStores = ([
                // Posts created by this client
                'mine',
                // Replies to the user's posts that have already been seen
                'seen',
                // Posts that the user has viewed or scrolled past
                'seenPost',
                // Posts hidden by client
                'hidden',
            ]);

            window.handle_db_error = function(e)
            {
                Module._handle_db_error(e.toString());
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

void load_post_ids(const std::unordered_set<uint64_t>& threads)
{
    if (!threads.size() || has_errored) {
        return;
    }

    // Copy to vector, so we can pass it to JS
    const std::vector<uint64_t> vec(threads.begin(), threads.end());

    EM_ASM_INT(
        {
            var left = $1 * postStores.length;

            for (var i = 0; i < $1; i++) {
                var id = getValue($0 + i * 8, 'i64');
                for (var j = 0; j < postStores.length; j++) {
                    read(id, j, postStores[j]);
                }
            }

            // Need to scope variables to function, because async. ES5 a
            // shit.
            function read(op, typ, name)
            {
                var ids = new Module.VectorUint64();
                var t = db.transaction(name, "readonly");
                t.onerror = handle_db_error;

                var range = IDBKeyRange.bound(op, op);
                var req = t.objectStore(name).index("op").openCursor(range);
                req.onerror = handle_db_error;
                req.onsuccess = function(event)
                {
                    var cursor = event.target.result;
                    if (cursor) {
                        ids.push_back(cursor.value.id);
                        cursor.continue();
                    } else {
                        Module.add_to_storage(typ, ids);
                        if (--left == 0) {
                            Module.render_page();
                        }
                    }
                };
            }
        },
        vec.data(), vec.size());
}

void handle_db_error(std::string err)
{
    has_errored = true;
    EM_ASM_INT({ console.error(Pointer_stringify($0)); }, err.c_str());
}

void db_is_ready()
{
    load_post_ids(*threads_to_load);
    delete threads_to_load;
}

EMSCRIPTEN_BINDINGS(module_db)
{
    emscripten::function("_handle_db_error", &handle_db_error);
    emscripten::function("db_is_ready", &db_is_ready);
}
