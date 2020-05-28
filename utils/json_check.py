# Copyright © 2020 Pavel Tisnovsky
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Simple checker of all JSONs in the given directory (usually repository)."""

from pathlib import Path
from json import load
from sys import exit
from os import popen
from argparse import ArgumentParser


def read_control_code(operation):
    """Try to execute tput to read control code for selected operation."""
    return popen("tput " + operation, "r").readline()


def check_jsons(verbose, directory):
    """Check all JSON files found in current directory and all subdirectories."""
    passes = 0
    failures = 0

    files = list(Path(directory).rglob("*.json"))

    for file in files:
        try:
            with file.open() as fin:
                obj = load(fin)
                if verbose is not None:
                    print("{} is valid".format(file))

                passes += 1
        except ValueError as e:
            print("{} is invalid".format(file))
            failures += 1
            print(e)

    return passes, failures


def display_report(passes, failures, nocolors):
    """Display report about number of passes and failures."""
    red_background = green_background = magenta_background = no_color = ""
    if not nocolors:
        red_background = read_control_code("setab 1")
        green_background = read_control_code("setab 2")
        magenta_background = read_control_code("setab 5")
        no_color = read_control_code("sgr0")

    if failures == 0:
        if passes == 0:
            print("{}[WARN]{}: no JSON files detected".format(magenta_background, no_color))
        else:
            print("{}[OK]{}: all JSONs have proper format".format(green_background, no_color))
    else:
        print("{}[FAIL]{}: invalid JSON(s) detected".format(red_background, no_color))

    print("{} passes".format(passes))
    print("{} failures".format(failures))


def main():
    """Entry point to this tool."""
    parser = ArgumentParser()
    parser.add_argument("-v", "--verbose", dest="verbose", help="make it verbose",
                        action="store_true", default=None)
    parser.add_argument("-n", "--no-colors", dest="nocolors", help="disable color output",
                        action="store_true", default=None)
    parser.add_argument("-d", "--directory", dest="directory",
                        help="directory with JSON files to check",
                        action="store", default=".")
    args = parser.parse_args()

    passes, failures = check_jsons(args.verbose, args.directory)
    display_report(passes, failures, args.nocolors)

    if failures > 0:
        exit(1)


if __name__ == "__main__":
    main()
