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
import os


def read_control_code(operation):
    """Try to execute tput to read control code for selected operation."""
    return popen("tput " + operation, "r").readline()


def check_jsons(verbose, directory):
    """Check all JSON files found in current directory and all subdirectories."""
    # Reset counters with number of passes and number of failures.
    passes = 0
    failures = 0

    # Find all files in current directory and subdirectories with `*.json`
    # extension. Files are found recursivelly.
    files = list(Path(directory).rglob("*.json"))

    # Iterate over all files found by previous command.
    for file in files:
        try:
            # If the file can be opened and loaded as JSON, everything is fine.
            with file.open() as fin:
                obj = load(fin)
                if verbose is not None:
                    print("{} is valid".format(file))

                passes += 1
        except ValueError as e:
            # There are several reasons and possibilities why the file can'b be
            # read as JSON.
            print("{} is invalid".format(file))
            failures += 1
            print(e)

    # Just the counters needs to be returned because all other informations
    # about problems have been displayed already.
    return passes, failures


def display_report(passes, failures, nocolors):
    """Display report about number of passes and failures."""
    # First of all, we need to setup colors to be displayed on terminal. Colors
    # are displayed by using terminal escape control codes. When colors are not
    # enabled on command line, simply we can simply use empty strings instead
    # of real color escape codes.
    red_background = green_background = magenta_background = no_color = ""

    # If colors are enabled by command line parameter, use control sequence
    # returned by `tput` command.
    if not nocolors:
        red_background = read_control_code("setab 1")
        green_background = read_control_code("setab 2")
        magenta_background = read_control_code("setab 5")
        no_color = read_control_code("sgr0")

    # There are four possible outcomes:
    # 1. no JSON files has been found
    # 2. all files are ok
    # 3. none of JSON files can be read and parsed
    # 4. some files can be read and parsed, some can not
    if failures == 0:
        if passes == 0:
            print("{}[WARN]{}: no JSON files detected".format(magenta_background, no_color))
        else:
            print("{}[OK]{}: all JSONs have proper format".format(green_background, no_color))
    else:
        print("{}[FAIL]{}: invalid JSON(s) detected".format(red_background, no_color))

    # Print just number of passes and failures at the end, as this information
    # can be processed on CI.
    print("{} passes".format(passes))
    print("{} failures".format(failures))


def main():
    """Entry point to this tool."""
    # First of all, we need to specify all command line flags that are
    # recognized by this tool.
    parser = ArgumentParser()
    parser.add_argument("-v", "--verbose", dest="verbose", help="make it verbose",
                        action="store_true", default=None)
    parser.add_argument("-n", "--no-colors", dest="nocolors", help="disable color output",
                        action="store_true", default=None)
    parser.add_argument("-d", "--directory", dest="directory",
                        help="directory with JSON files to check",
                        action="store", default=".")
    # Now it is time to parse flags, check the actual content of command line
    # and fill in the object named `args`.
    args = parser.parse_args()

    # Check all JSON files, display problems, and get counters with number of
    # passes and failures.
    verbose = args.verbose or "VERBOSE" in os.environ
    passes, failures = check_jsons(verbose, args.directory)

    # Display detailed report and summary as well.
    display_report(passes, failures, args.nocolors)

    # If any error is found, return with exit code check to non-zero value.
    if failures > 0:
        exit(1)


# If this script is started from command line, run the `main` function which is
# entry point to the processing.
if __name__ == "__main__":
    main()
