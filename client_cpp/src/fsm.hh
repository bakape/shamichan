#pragma once

#include <functional>
#include <map>
#include <type_traits>
#include <unordered_map>
#include <utility>
#include <vector>

// Finite State Machine
template <class S, class E> class FSM {
    static_assert(std::is_enum<S>::value, "must be an enum type");
    static_assert(std::is_enum<E>::value, "must be an enum type");

public:
    // Defines a type transition and executes arbitrary code
    typedef std::function<S()> Handler;

    // Create a new FSM with the supplied start state
    FSM(S state)
        : _state(state)
    {
    }

    // Return the current state
    S state() const { return _state; };

    // Assign a handler to be execute on arrival to a new state
    void on(S state, std::function<void()> fn)
    {
        state_handlers[state].push_back(fn);
    }

    // Like on, but handler is removed after execution
    void once(S state, std::function<void()> fn)
    {
        once_handlers[state].push_back(fn);
    }

    // Specify state transition and a handler to execute on it. The handler must
    // return the next state of FSM.
    void act(S start, E event, Handler fn)
    {
        transitions[{ start, event }] = fn;
    }

    // Specify an event and handler, that will execute, when this event is
    // fired, on any state
    void wild_act(E event, Handler fn) { wilds[event] = fn; }

    // Feed an event to the FSM
    void feed(E event)
    {
        S result;

        if (wilds.count(event)) {
            result = wilds[event]();
        } else {
            const std::pair<S, E> key = { _state, event };
            if (!transitions.count(key)) {
                // Not registered - NOP
                return;
            }
            result = transitions[key]();
        }

        if (result == _state) {
            return;
        }
        for (auto& fn : state_handlers[result]) {
            fn();
        }
        for (auto& fn : once_handlers[result]) {
            fn();
        }
        once_handlers[result].clear();

        _state = result;
    }

private:
    // Current state
    S _state;

    struct EnumHasher {
        template <class T> size_t operator()(T t) const
        {
            return static_cast<size_t>(t);
        }
    };

    std::unordered_map<S, std::vector<std::function<void()>>, EnumHasher>
        // Handlers executed on arival to a new state
        state_handlers,
        // Handlers executed on arival to a new state, but only once
        once_handlers;

    // Functions to execute, when an event fires on a state
    std::map<std::pair<S, E>, Handler> transitions;

    // Functions to execute on an event no matter what state FSM is in
    std::unordered_map<E, Handler> wilds;
};
