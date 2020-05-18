---
layout: page
nav_order: 9
---
# Rules

The user has the ability to disable a rule/health check recommendation that they're not interested
in to stop it from showing in OCM. The user also has the ability to re-enable the rule, in case they
later become interested in it, or in the case of an accidental disable, for example.

This is made possible by using these two endpoints:
`clusters/{cluster}/rules/{rule_id}/disable`
`clusters/{cluster}/rules/{rule_id}/enable`

## Tutorial rule

Directory `rules/tutorial/` contains tutorial rule that is 'hit' by any cluster.

## Rules content

The generated cluster reports from Insights results contain three lists of rules that were either
__skipped__ (because of missing requirements, etc.), __passed__ (the rule got executed but no issue
was found), or __hit__ (the rule got executed and found the issue it was looking for) by the
cluster, where each rule is represented as a dictionary containing identifying information about the
rule itself.

The __hit__ rules are the rules that the customer is interested in and therefore the information
about them (their __content__) needs to be displayed in OCM. The content for the rules is present in
another repository, alongside the actual rule implementations, which is primarily maintained by the
support-facing team.
For that reason, the `insights-results-aggregator` is setup in a way, that the content we're
interested in is copied to the Docker image from the repository during image build and we have a
push webhook on master branch set up in that repository, signalling our app to be rebuilt.

This content is then processed upon application start-up, correctly parsed by the package `content`
and saved into the database (see DB structure).

When a request for a cluster report comes from OCM, the report is parsed (TODO: parse reports only
once when consuming them) and content for all the hit rules is returned.

### Local environment with rules content

The rules content parser is configured by default to expect the content in a root directory
`/rules-content`.
This can be changed either by an environment variable `INSIGHTS_RESULTS_AGGREGATOR__CONTENT__PATH`
or by modifying the config file entry:

```toml
[content]
path = "/rules-content"
```

To get the latest rules content locally, you can `make rules_content`, which just runs the script
`update_rules_content.sh` mimicking the Dockerfile behavior (NOTE: you need to be in RH VPN to be
able to access that repository, but it is not private). The script copies the content into a
`.gitignored` folder `rules-content`, so all that's necessary is to change the expected path.
