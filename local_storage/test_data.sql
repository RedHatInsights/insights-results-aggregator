-- Delete from report
DELETE FROM report;

-- Insert into report
INSERT INTO report (org_id, cluster, report, reported_at, last_checked_at)
VALUES
    (1, '00000000-0000-0000-0000-000000000000', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (1, '00000000-0000-0000-ffff-000000000000', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (1, '00000000-0000-0000-0000-ffffffffffff', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, '00000000-ffff-0000-0000-000000000000', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (2, '00000000-0000-ffff-0000-000000000000', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (3, 'aaaaaaaa-0000-0000-0000-000000000000', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (3, 'addddddd-0000-0000-0000-000000000000', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (4, 'addddddd-bbbb-0000-0000-000000000000', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (4, 'addddddd-bbbb-cccc-0000-000000000000', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

-- Delete from recommendation
DELETE FROM recommendation;

-- Insert into recommendation
INSERT INTO recommendation (org_id, cluster_id, rule_fqdn, error_key, rule_id, created_at)
VALUES
    (1, '11111111-1111-1111-1111-111111111111', 'ccx_rules_ocp.external.rules.node_installer_degraded', 'ek1', 'ccx_rules_ocp.external.rules.node_installer_degraded|ek1', CURRENT_TIMESTAMP),
    (2, '22222222-2222-2222-2222-222222222222', 'ccx_rules_ocp.external.rules.node_installer_degraded', 'ek1', 'ccx_rules_ocp.external.rules.node_installer_degraded|ek1', CURRENT_TIMESTAMP),
    (3, '33333333-3333-3333-3333-333333333333', 'ccx_rules_ocp.external.rules.node_installer_degraded', 'ek1', 'ccx_rules_ocp.external.rules.node_installer_degraded|ek1', CURRENT_TIMESTAMP);

-- Delete from report_info
DELETE FROM report_info;

-- Insert into report_info
INSERT INTO report_info (org_id, cluster_id, version_info)
VALUES
    (1, '11111111-1111-1111-1111-111111111111', '1.0'),
    (2, '22222222-2222-2222-2222-222222222222', '');

-- Delete from cluster_rule_user_feedback
DELETE FROM cluster_rule_user_feedback;

