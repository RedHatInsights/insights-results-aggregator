#!/usr/bin/env bash
# Copyright 2020 Red Hat, Inc
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

_rules_content_repo="git@gitlab.cee.redhat.com:ccx/ccx-rules-ocp.git"

_script_dir="$(cd "$(dirname "$0")"; pwd)"
_content_dir="${_script_dir}/rules/content"

function clone_fresh_repo() {
    echo "Cloning git repository"
    git clone "$_rules_content_repo" "$_content_dir" &> /dev/null
    
    if [ $? -eq 0 ]
    then
        echo "Git clone successful"
    else
        echo "Git clone unsuccessful. Exiting."
        exit 1
    fi
}

# checks the presence of index.lock file which prevents git operations
function repo_lock_check() {
    if [ -f "$_content_dir/.git/index.lock" ]
    then
        echo "Local 'content' repository is locked. If no other git process is running, remove the index.lock file. Exiting."
        exit 1
    fi
}

# check if the repository in rules/content is the correct one
function is_rules_content_repo() {
    content_git_repo="$(git -C "$_content_dir" config --get remote.origin.url)"
    if [ "$_rules_content_repo" == "$content_git_repo" ]
    then
        return 0
    fi
    return 1
}

# fetches the remote repo, compares commit hashes and pulls if necessary
function fetch_and_update() {
    git -C "$_content_dir" checkout master &> /dev/null
    git -C "$_content_dir" fetch origin master &> /dev/null

    last_local_commit="$(git -C "$_content_dir" rev-parse HEAD)"
    last_remote_commit="$(git -C "$_content_dir" rev-parse origin/master)"
    
    if [ "$last_local_commit" == "$last_remote_commit" ]
    then
        echo "No new changes in remote repository. Exiting peacefully."
        exit 0
    fi

    git -C "$_content_dir" pull origin master &> /dev/null

    echo "Content repository updated successfully."
}

# check if the repo isn't locked by another git process
repo_lock_check

if [ ! -d "$_content_dir" ]
then
    echo "Creating content directory"
    mkdir "$_content_dir"
    clone_fresh_repo
else
    if is_rules_content_repo
    then
        echo "Correct repository found. Attempting to update."
        fetch_and_update
    else
        echo "Content git repository not found in content directory. Exiting."
        exit 1
    fi
fi
