#!/usr/bin/env bash

# Copies config file, if none
if [[ ! -a config/config.json ]]; then
    cp config/defaults.json config/config.json
fi
