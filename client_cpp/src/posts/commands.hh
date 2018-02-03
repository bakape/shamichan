#pragma once

// Offset between the client's and server's clocks
extern long server_time_offset;

// Rerender all posts, that contain syncwatches, if any
void rerender_syncwatches();
