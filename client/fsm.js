// Finite State Machine
export default class FSM {
    // Create a new finite state machine
    constructor(start) {
        this.state = start
        this.spec = {
            acts: {},
            ons: {},
            wilds: {},
            preflights: {}
        }
    }

    // Clone the current FSM
    clone() {
        const second = new FSM(this.state)
        second.spec = this.spec
        return second
    }

    // Assign a handler to be execute on arrival to a new state
    on(key, f) {
        const ons = this.spec.ons[key]
        if (ons) {
            ons.push(f)
        } else {
            this.spec.ons[key] = [f]
        }
    }

    // Assign sanity check to perform before transition to a new state
    preflight(key, f) {
        const pres = this.spec.preflights[key]
        if (pres) {
            pres.push(f)
        } else {
            this.spec.preflights[key] = [f]
        }
    }

    // Specify transition and an optional handler to execute on it
    act(trans_spec, on_func) {
        const halves = trans_spec.split('->')
        if (halves.length != 2) {
            throw new Error("Bad FSM spec: " + trans_spec)
        }
        const parts = halves[0].split(','),
            dest = halves[1].match(/^\s*(\w+)\s*$/)[1]
        let tok
        for (let i = parts.length - 1; i >= 0; i--) {
            const part = parts[i],
                m = part.match(/^\s*(\*|\w+)\s*(?:\+\s*(\w+)\s*)?$/)
            if (!m) {
                throw new Error("Bad FSM spec portion: " + part)
            }
            if (m[2]) {
                tok = m[2]
            }
            if (!tok) {
                throw new Error("Tokenless FSM action: " + part)
            }
            const src = m[1]
            if (src == '*') {
                this.spec.wilds[tok] = dest
            } else {
                let acts = this.spec.acts[src]
                if (!acts) {
                    this.spec.acts[src] = acts = {}
                }
                acts[tok] = dest
            }
        }
        if (on_func) {
            this.on(dest, on_func)
        }
    }

    // Transition the FSM to a new state
    feed(ev, param) {
        const {spec} = this,
            from = this.state,
            acts = spec.acts[from],
            to = (acts && acts[ev]) || spec.wilds[ev]
        if (to && from != to) {
            const ps = spec.preflights[to]
            for (let i = 0; ps && i < ps.length; i++) {
                if (!ps[i].call(this, param)) {
                    return false
                }
            }
            this.state = to
            const fs = spec.ons[to]
            for (let i = 0; fs && i < fs.length; i++) {
                fs[i].call(this, param)
            }
        }
        return true
    }

    // Returns a function that executes FSM.prototype.feed with the suplied
    // argument
    feeder(ev) {
        return param =>  this.feed(ev, param)
    }
}
