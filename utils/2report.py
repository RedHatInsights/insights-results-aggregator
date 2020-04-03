#!/usr/bin/env python3

# Copyright © 2020 Red Hat, Inc.
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

"""Converts outputs from OCP rule engine into proper reports.

All input files that have filename 's_*.json' (usually anonymized
outputs from OCP rule engine') are converted into proper 'report'
that can be:

    1. Published into Kafka topic
    2. Stored directly into aggregator database

It is done by inserting organization ID, clusterName and lastChecked
attributes and by rearranging output structure. Output files will
have following names: 'r_*.json'.
"""

import json
import sys
import datetime
from os import listdir
from os.path import isfile, join

files = [f for f in listdir(".") if isfile(join(".", f))]

if len(sys.argv) < 3:
    print("Usage: 2report.py org_id cluster_id")

orgID = sys.argv[1]
clusterName = sys.argv[2]
lastChecked = datetime.datetime.utcnow().isoformat() + "Z"

def remove_internal_rules(data, key, selector):
    if "reports" in data:
        if key in data:
            reports = data[key]
            new = []
            for report in reports:
                if not report[selector].startswith("ccx_rules_ocp.internal."):
                    print("adding", report[selector])
                    new.append(report)
            data[key] = new

for filename in files:
    if filename.startswith("s_") and filename.endswith(".json"):
        with open(filename) as fin:
            data = json.load(fin)
            remove_internal_rules(data, "reports", "component")
            remove_internal_rules(data, "pass", "component")
            remove_internal_rules(data, "skips", "rule_fqdn") # oh my...

            outfilename = "r_" + filename[2:]
            report = {}
            report["OrgID"] = int(orgID)
            report["ClusterName"] = clusterName
            report["LastChecked"] = lastChecked
            report["Report"] = data

            with open(outfilename, "w") as fout:
                json.dump(report, fout, indent=4)
