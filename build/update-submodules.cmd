@echo off
git submodule init
git submodule update
git submodule foreach git pull origin master
git commit -am "chore: bump submodules"
git push