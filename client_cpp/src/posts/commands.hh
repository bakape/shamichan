#pragma once

// Offset between the client's and server's clocks
inline long server_time_offset = 0;

// Rerender all posts, that contain syncwatches, if any
void rerender_syncwatches();
