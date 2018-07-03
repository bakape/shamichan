#pragma once

#include <functional>
#include <string>

// Callback executed after finishing or failing an HTTP request
typedef std::function<void(unsigned short, std::string)> HTTPCallback;

// Run an HTTP request on URL and execute cb on result or error
void http_request(std::string url, HTTPCallback cb);
