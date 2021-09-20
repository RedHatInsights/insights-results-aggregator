# Copyright © 2021 Pavel Tisnovsky
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

"""
Simple script to fill in rule disable tables with test data.

usage: fill_in_disable_rule_tables.py [-h] [-v] [-f] [-p PROBABILITY]
                                      [-d DATABASE] [-U USER] [-P PASSWORD]

optional arguments:
  -h, --help            show this help message and exit
  -v, --verbose         make it verbose
  -f, --feedback        fill-in cluster_user_rule_disable_feedback
  -p PROBABILITY, --probability PROBABILITY
                        probability of rule to be disabled
  -d DATABASE, --database DATABASE
                        database name
  -U USER, --user USER  user in database
  -P PASSWORD, --password PASSWORD
                        password to connect to database
"""

from argparse import ArgumentParser
from random import random, choice
from subprocess import Popen
from datetime import datetime
import psycopg2


# default values
DEFAULT_RULE_DISABLE_PROBABILITY = 100

ACCOUNT_IDS = (1, 2, 3, 4)

CLUSTER_IDS = (
    "00000001-624a-49a5-bab8-4fdc5e51a266",
    "00000001-624a-49a5-bab8-4fdc5e51a267",
    "00000001-624a-49a5-bab8-4fdc5e51a268",
    "00000001-624a-49a5-bab8-4fdc5e51a269",
    "00000001-624a-49a5-bab8-4fdc5e51a26a",
    "00000001-624a-49a5-bab8-4fdc5e51a26b",
    "00000001-624a-49a5-bab8-4fdc5e51a26c",
    "00000001-624a-49a5-bab8-4fdc5e51a26d",
    "00000001-624a-49a5-bab8-4fdc5e51a26e",
    "00000001-624a-49a5-bab8-4fdc5e51a26f",
    "00000001-6577-4e80-85e7-697cb646ff37",
    "00000001-8933-4a3a-8634-3328fe806e08",
    "00000001-8d6a-43cc-b82c-7007664bdf69",
    "00000002-624a-49a5-bab8-4fdc5e51a266",
    "00000002-6577-4e80-85e7-697cb646ff37",
    "00000002-8933-4a3a-8634-3328fe806e08",
    "00000003-8933-4a3a-8634-3328fe806e08",
    "00000003-8d6a-43cc-b82c-7007664bdf69",
    "34c3ecc5-624a-49a5-bab8-4fdc5e51a266",
    "34c3ecc5-624a-49a5-bab8-4fdc5e51a267",
    "34c3ecc5-624a-49a5-bab8-4fdc5e51a268",
    "34c3ecc5-624a-49a5-bab8-4fdc5e51a269",
    "34c3ecc5-624a-49a5-bab8-4fdc5e51a26a",
    "34c3ecc5-624a-49a5-bab8-4fdc5e51a26b",
    "34c3ecc5-624a-49a5-bab8-4fdc5e51a26c",
    "34c3ecc5-624a-49a5-bab8-4fdc5e51a26d",
    "34c3ecc5-624a-49a5-bab8-4fdc5e51a26e",
    "34c3ecc5-624a-49a5-bab8-4fdc5e51a26f",
    "74ae54aa-6577-4e80-85e7-697cb646ff37",
    "a7467445-8d6a-43cc-b82c-7007664bdf69",
    "ee7d2bf4-8933-4a3a-8634-3328fe806e08")

RULE_SELECTORS = (
    ("ccx_rules_ocp.external.rules.ccxdev_auxiliary_rule", "CCXDEV_E2E_TEST_RULE"),
    ("ccx_rules_ocp.external.bug_rules.bug_1821905.report", "BUGZILLA_BUG_1821905"),
    ("ccx_rules_ocp.external.rules.nodes_requirements_check.report",
        "NODES_MINIMUM_REQUIREMENTS_NOT_MET"),
    ("ccx_rules_ocp.external.bug_rules.bug_1766907.report", "BUGZILLA_BUG_1766907"),
    ("ccx_rules_ocp.external.rules.nodes_kubelet_version_check.report", "NODE_KUBELET_VERSION"),
    ("ccx_rules_ocp.external.rules.samples_op_failed_image_import_check.report",
        "SAMPLES_FAILED_IMAGE_IMPORT_ERR"))


def main():
    """Entry point to this tool."""
    # First of all, we need to specify all command line flags that are
    # recognized by this tool.
    parser = ArgumentParser()
    parser.add_argument("-v", "--verbose", dest="verbose",
                        help="make it verbose",
                        action="store_true", default=None)
    parser.add_argument("-f", "--feedback", dest="feedback",
                        help="fill-in cluster_user_rule_disable_feedback",
                        action="store_true", default=None)
    parser.add_argument("-p", "--probability", dest="probability",
                        help="probability of rule to be disabled",
                        type=int, default=DEFAULT_RULE_DISABLE_PROBABILITY)
    parser.add_argument("-d", "--database", dest="database",
                        help="database name",
                        type=str, default="aggregator")
    parser.add_argument("-U", "--user", dest="user",
                        help="user in database",
                        type=str, default="postgres")
    parser.add_argument("-P", "--password", dest="password",
                        help="password to connect to database",
                        type=str, default="postgres")

    # Now it is time to parse flags, check the actual content of command line
    # and fill in the object stored in variable named `args`.
    args = parser.parse_args()


# If this script is started from command line, run the `main` function
# which represents entry point to the processing.
if __name__ == "__main__":
    """Entry point to this tool."""
    main()
