--PRAGMA foreign_keys = ON;
--CREATE SCHEMA public;

create table report (
    org_id      integer not null,
    cluster     varchar not null unique,
    report      varchar not null,
    reported_at timestamp,
    last_checked_at timestamp,
    PRIMARY KEY(org_id, cluster)
);
