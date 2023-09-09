#!/bin/bash
git submodule init
git submodule update
cd sealdice-builtins && git pull origin master && cd ..
cd sealdice-ui && git pull origin master && cd ..
cd sealdice-android && git pull origin master && cd ..
cd go-cqhttp && git pull origin master && cd ..
cd sealdice-core && git pull origin master && cd ..
