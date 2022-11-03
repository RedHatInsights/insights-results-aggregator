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
  -r, --rule-disable    fill-in rule_disable table
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

ACCOUNT_IDS = (1, 2, 3, 4, 5, 6)

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
    "ee7d2bf4-8933-4a3a-8634-3328fe806e08",
)

RULE_SELECTORS = (
    ("ccx_rules_ocp.external.test.rule_A", "ERROR_KEY_1"),
    ("ccx_rules_ocp.external.test.rule_A", "ERROR_KEY_2"),
    ("ccx_rules_ocp.external.test.rule_A", "ERROR_KEY_3"),
    ("ccx_rules_ocp.external.test.rule_B", "ERROR_KEY_1"),
    ("ccx_rules_ocp.external.test.rule_B", "ERROR_KEY_2"),
    ("ccx_rules_ocp.external.test.rule_B", "ERROR_KEY_3"),
    ("ccx_rules_ocp.external.rules.ccxdev_auxiliary_rule", "CCXDEV_E2E_TEST_RULE"),
    ("ccx_rules_ocp.external.bug_rules.bug_1821905.report", "BUGZILLA_BUG_1821905"),
    (
        "ccx_rules_ocp.external.rules.nodes_requirements_check.report",
        "NODES_MINIMUM_REQUIREMENTS_NOT_MET",
    ),
    ("ccx_rules_ocp.external.bug_rules.bug_1766907.report", "BUGZILLA_BUG_1766907"),
    (
        "ccx_rules_ocp.external.rules.nodes_kubelet_version_check.report",
        "NODE_KUBELET_VERSION",
    ),
    (
        "ccx_rules_ocp.external.rules.samples_op_failed_image_import_check.report",
        "SAMPLES_FAILED_IMAGE_IMPORT_ERR",
    ),
)


def fill_in_database_for_clusters(connection, verbose, feedback, probability):
    """Fill-in database with test data assigned to clusters."""
    # add new info about rule disable, but only with some probability.
    for cluster_id in CLUSTER_IDS:
        if verbose:
            print("\tCluster ID {}".format(cluster_id))
        for rule_selector in RULE_SELECTORS:
            if random() * 100 < probability:
                account_id = choice(ACCOUNT_IDS)
                if verbose:
                    print("\t\tAccount ID: {}".format(account_id))
                    print("\t\tAdding new rule disable info: {}".format(rule_selector))
                insert_into_db_for_clusters(
                    connection, account_id, cluster_id, rule_selector, feedback
                )


def fill_in_database_system_wide_rule_disable(connection, verbose, probability):
    """Fill-in database with system-wide test data."""
    # add new info about rule disable, but only with some probability.
    for rule_selector in RULE_SELECTORS:
        if random() * 100 < probability:
            account_id = choice(ACCOUNT_IDS)
            if verbose:
                print("\t\tAccount ID: {}".format(account_id))
                print("\t\tAdding new rule disable info: {}".format(rule_selector))
            insert_into_db_system_wide_disable(connection, account_id, rule_selector)


def check_if_postgres_is_running():
    """Check if Postgresql service is active."""
    p = Popen(["systemctl", "is-active", "--quiet", "postgresql"])
    assert p is not None

    # interact with the process:
    p.communicate()

    # check the return code
    assert (
        p.returncode == 0
    ), "Postgresql service not running: got return code {code}".format(
        code=p.returncode
    )


def connect_to_database(database, user, password):
    """Perform connection to selected database."""
    connection_string = "dbname={} user={} password={}".format(database, user, password)
    return psycopg2.connect(connection_string)


def disconnect_from_database(connection):
    """Close the connection to database."""
    connection.close()


def insert_into_db_for_clusters(
    connection, account_id, cluster_id, rule_selector, feedback
):
    """Insert new record(s) into database."""
    cursor = connection.cursor()

    try:
        # try to perform insert statement
        insertStatement = """INSERT INTO cluster_rule_toggle
                             (cluster_id, rule_id, error_key, disabled, disabled_at,
                             updated_at, org_id)
                             VALUES(%s, %s, %s, 1, %s, %s, 1);"""

        # generate timestamp to be stored in database
        timestamp = datetime.now().strftime("%Y-%m-%d")

        # insert in transaction
        cursor.execute(
            insertStatement,
            (
                cluster_id,
                rule_selector[0],
                rule_selector[1],
                timestamp,
                timestamp,
            ),
        )

        if feedback:
            # perform insert into cluster_user_rule_disable_feedback
            insertStatement = """INSERT INTO cluster_user_rule_disable_feedback
                                 (cluster_id, user_id, rule_id, error_key, message, added_at,
                                 updated_at, org_id)
                                 VALUES(%s, %s, %s, %s, %s, %s, %s, %s);"""

            # some nice feedback from user
            message = "Rule {}|{} for cluster {} disabled by {}".format(
                rule_selector[0], rule_selector[1], cluster_id, account_id, 1
            )

            # insert in transaction
            cursor.execute(
                insertStatement,
                (
                    cluster_id,
                    account_id,
                    rule_selector[0],
                    rule_selector[1],
                    message,
                    timestamp,
                    timestamp,
                ),
            )

        connection.commit()
    except Exception as e:
        connection.rollback()
        raise e


def insert_into_db_system_wide_disable(connection, account_id, rule_selector):
    """Insert new record(s) into database."""
    cursor = connection.cursor()

    try:
        # try to perform insert statement
        insertStatement = """INSERT INTO rule_disable
                             (rule_id, error_key, org_id, user_id, created_at,
                             updated_at, justification)
                             VALUES(%s, %s, 1, %s, %s, %s, %s);"""

        # some nice feedback from user
        justification = "Rule {}|{} has been disabled by {}".format(
            rule_selector[0], rule_selector[1], account_id
        )

        # generate timestamp to be stored in database
        timestamp = datetime.now().strftime("%Y-%m-%d")

        # insert in transaction
        cursor.execute(
            insertStatement,
            (
                rule_selector[0],
                rule_selector[1],
                account_id,
                timestamp,
                timestamp,
                justification,
            ),
        )

        connection.commit()
    except Exception as e:
        connection.rollback()
        raise e


def main():
    """Entry point to this tool."""
    # First of all, we need to specify all command line flags that are
    # recognized by this tool.
    parser = ArgumentParser()
    parser.add_argument(
        "-v",
        "--verbose",
        dest="verbose",
        help="make it verbose",
        action="store_true",
        default=None,
    )
    parser.add_argument(
        "-f",
        "--feedback",
        dest="feedback",
        help="fill-in cluster_user_rule_disable_feedback",
        action="store_true",
        default=None,
    )
    parser.add_argument(
        "-r",
        "--rule-disable",
        dest="rule_disable",
        help="fill-in rule_disable table",
        action="store_true",
        default=None,
    )
    parser.add_argument(
        "-p",
        "--probability",
        dest="probability",
        help="probability of rule to be disabled",
        type=int,
        default=DEFAULT_RULE_DISABLE_PROBABILITY,
    )
    parser.add_argument(
        "-d",
        "--database",
        dest="database",
        help="database name",
        type=str,
        default="aggregator",
    )
    parser.add_argument(
        "-U",
        "--user",
        dest="user",
        help="user in database",
        type=str,
        default="postgres",
    )
    parser.add_argument(
        "-P",
        "--password",
        dest="password",
        help="password to connect to database",
        type=str,
        default="postgres",
    )

    # Now it is time to parse flags, check the actual content of command line
    # and fill in the object stored in variable named `args`.
    args = parser.parse_args()

    # try to connect to database
    check_if_postgres_is_running()
    connection = connect_to_database(args.database, args.user, args.password)
    assert connection is not None

    # fill the database by test data
    if args.rule_disable:
        fill_in_database_system_wide_rule_disable(
            connection, args.verbose, args.probability
        )
    else:
        fill_in_database_for_clusters(
            connection, args.verbose, args.feedback, args.probability
        )

    # everything's seems ok, let's disconnect
    disconnect_from_database(connection)


# If this script is started from command line, run the `main` function
# which represents entry point to the processing.
if __name__ == "__main__":
    """Entry point to this tool."""
    main()
