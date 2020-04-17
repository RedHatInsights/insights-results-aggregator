#!/bin/bash
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

"""Display statistic about rules that really 'hit' problems on clusters."""

import collections
import json
from os import listdir
from os.path import isfile, join

files = [f for f in listdir(".") if isfile(join(".", f))]

rule_names = (
    'ccx_rules_ocp.external.bug_rules.bug_1766907.report',
    'ccx_rules_ocp.external.bug_rules.bug_1798049.report',
    'ccx_rules_ocp.external.bug_rules.bug_1801300.report',
    'ccx_rules_ocp.external.bug_rules.bug_1802248.report',
    'ccx_rules_ocp.external.rules.cluster_wide_proxy_auth_check.report',
    'ccx_rules_ocp.external.rules.image_registry_no_volume_set_check.report',
    'ccx_rules_ocp.external.rules.nodes_kubelet_version_check.report',
    'ccx_rules_ocp.external.rules.nodes_requirements_check.report',
    'ccx_rules_ocp.external.rules.pods_crash_loop_check.report',
    'ccx_rules_ocp.internal.rules.certificates_expiration.report',
    'ccx_rules_ocp.internal.rules.certificates_info.report',
    'ccx_rules_ocp.internal.rules.certificates_validity.report',
    'ccx_rules_ocp.internal.rules.event_nfs_conf.report',
    'ccx_rules_ocp.internal.rules.machine_pool_check.report',
    'ccx_rules_ocp.internal.rules.machine_pool_info.report',
    'ccx_rules_ocp.internal.rules.machine_update_stuck.report',
    'ccx_rules_ocp.internal.rules.nodes_info.report',
    'ccx_rules_ocp.internal.rules.nodes_pressure_check.report',
    'ccx_rules_ocp.internal.rules.operators_check.report',
    'ccx_rules_ocp.internal.rules.pods_check_containers.report',
    'ccx_rules_ocp.internal.rules.pods_check.report',
    'ccx_rules_ocp.internal.rules.version_check.report',
    'ccx_rules_ocp.internal.rules.version_forced.report',
    'ccx_rules_ocp.internal.rules.version_retarget.report',
    'ccx_rules_ocp.internal.telemetry_rules.support_check.report',
    'ccx_rules_ocp.internal.telemetry_rules.version_check.report',
    'ccx_rules_ocp.internal.telemetry_rules.version_info.report',
    'ccx_rules_ocp.ocs.operator_phase_check.report',
    'ccx_rules_ocp.ocs.pvc_phase_check.report',
)

passed_cnt = collections.Counter()
skipped_cnt = collections.Counter()
reported_cnt = collections.Counter()

files = files[:10]

for filename in files:
    if filename.endswith(".json"):
        with open(filename) as fin:
            data = json.load(fin)
            if "info" in data:
                infolist = data["info"]
                cluster = None
                for info in infolist:
                    if info["key"] == "GRAFANA_LINK":
                        cluster = info["details"]["cluster_id"]
                if cluster is not None:
                    if "pass" in data:
                        passed = data["pass"]
                        for p in passed:
                            rule = p["component"]
                            passed_cnt[rule] += 1
                    if "skips" in data:
                        skipped = data["skips"]
                        for s in skipped:
                            rule = s["rule_fqdn"]
                            skipped_cnt[rule] += 1
                    if "reports" in data:
                        reports = data["reports"]
                        for r in reports:
                            rule = p["component"]
                            reported_cnt[rule] += 1

print("Rule, passed, reported, skipped")

for rule in rule_names:
    print(rule, passed_cnt[rule], reported_cnt[rule], skipped_cnt[rule],
          sep=",")
