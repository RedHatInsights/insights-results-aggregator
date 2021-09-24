---
layout: page
nav_order: 12
---
# Utilities

Utilities are stored in `utils` subdirectory.

## `json_check.py`

Simple checker if all JSONs have the correct syntax (not scheme).

### Script usage

```text
usage: json_check.py [-h] [-v]

optional arguments:
  -h, --help     show this help message and exit
  -v, --verbose  make it verbose
  -n, --no-colors  disable color output
  -d DIRECTORY, --directory DIRECTORY
                        directory with JSON files to check
```

[Annotated source code](json_check.html)

## `fill_in_disable_rule_tables.py`

Simple script to fill in rule disable tables with test data.

### Script usage

```text
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
```

## `docgo.sh`

Generates documentation for all packages.

Please note that other utilities and tools have in available in separate repository
[RedHatInsights / insights-results-aggregator-utils](https://github.com/RedHatInsights/insights-results-aggregator-utils/).
More detailed description of these tools is
[available there](https://github.com/RedHatInsights/insights-results-aggregator-utils/blob/master/README.md).
