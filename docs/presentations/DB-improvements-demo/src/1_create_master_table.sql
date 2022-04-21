CREATE TABLE recommendation_master (
    org_id     integer NOT NULL,
    cluster_id character varying NOT NULL,
    rule_fqdn  text NOT NULL,
    error_key  character varying NOT NULL,
    rule_id    character varying,
    created_at timestamp without time zone
);
