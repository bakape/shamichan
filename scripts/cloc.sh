#! /bin/bash

cloc --exclude-dir=node_modules,target,target_tarpaulin,www --exclude-ext=json,js ./
