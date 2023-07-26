#!/bin/bash
# Copyright 2021 Red Hat, Inc
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

DOCFILES=$(find docs/packages -name '*.html' | sort)
cp docs/sources.tmpl.md docs/sources.md

for filepath in $DOCFILES; do
    text=${filepath#*/packages/}
    text=${text/.html/.go}
    link=.${filepath#docs}
    echo "* [$text]($link)" >> docs/sources.md
done

exit 0
