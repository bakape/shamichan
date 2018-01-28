#include "page/page.hh"
#include "state.hh"
#include "util.hh"
#include <cstdint>
#include <emscripten.h>
#include <emscripten/bind.h>
#include <string>
#include <unordered_set>
#include <vector>

const int db_version = 7;

// Database has errored and all future calls should be ignored
bool has_errored = false;

// Has completed or errored out of loading the database at least once
bool has_loaded = false;

void open_db(WaitGroup* wg)
{
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
                Module._handle_db_error(e.toString(), $1);
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

                Module.db_is_ready($1);

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
                                          .index('expires')
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
        db_version, reinterpret_cast<int>(wg));
}

void load_post_ids()
{
    if (!threads->size() || has_errored) {
        render_page();
        return;
    }

    // Map to vector, so we can pass it to JS
    std::vector<unsigned long> ids;
    ids.reserve(threads->size());
    for (auto && [ id, _ ] : *threads) {
        ids.push_back(id);
    }

    EM_ASM_INT(
        {
            var left = 0;

            for (var i = 0; i < $1; i++) {
                var id = getValue($0 + i * 8, 'i64');
                left += postStores.length;
                for (var j = 0; j < postStores.length; j++) {
                    read(id, j, postStores[j]);
                }
            }

            // Need to scope variables to function, because async. ES5 a
            // shit.
            function read(op, typ, name)
            {
                var ids = new Module.VectorUint64();
                var t = db.transaction(name, 'readonly');
                t.onerror = handle_db_error;

                var range = IDBKeyRange.bound(op, op);
                var req = t.objectStore(name).index('op').openCursor(range);
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
        ids.data(), ids.size());
}

// Signals the database is ready. Called from the JS side.
static void db_is_ready(int wg)
{
    has_loaded = true;
    reinterpret_cast<WaitGroup*>(wg)->done();
}

// Handle a database error
static void handle_db_error(std::string err, int wg)
{
    console::error(err);
    if (!has_loaded) {
        db_is_ready(wg);
    }
}

EMSCRIPTEN_BINDINGS(module_db)
{
    emscripten::function("_handle_db_error", &handle_db_error);
    emscripten::function("db_is_ready", &db_is_ready);
}
