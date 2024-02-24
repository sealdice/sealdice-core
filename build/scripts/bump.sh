#!/usr/bin/env bash

LOCAL_BUILD_REPO='/home/ubuntu/project/sealdice-extra/build'
SEALDICE_REMOTE_NAME='origin'

set -ex

PWD_NOW=$(pwd)
cd "$LOCAL_BUILD_REPO"

git fetch --all --prune

git checkout dev
git reset "$SEALDICE_REMOTE_NAME/dev" --hard
./update-submodules.sh || true

# git checkout next
# git reset origin/next --hard
# ./update-submodules.sh || true

cd "${PWD_NOW}"
unset PWD_NOW

set +x

repo='sealdice/sealdice-build'
sleep 10s

runID=$(gh -R "$repo" run list -w 'Auto Build' -b 'dev' \
  -s 'in_progress' -L 1 --json databaseId -q '.[0].databaseId')

if [[ -z $runID ]]; then
  echo "No build in progress"
else
  gh -R sealdice/sealdice-build run watch "$runID"
fi

unset runID
unset repo
