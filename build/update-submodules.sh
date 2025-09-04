#!/bin/bash
git pull
git submodule init
git submodule update
git submodule foreach 'git fetch origin --prune && git reset --hard origin/master'
git commit -am "chore: bump submodules"
git push
