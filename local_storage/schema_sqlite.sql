create table report (
    org_id      integer not null,
    cluster     varchar not null unique,
    report      varchar not null,
    reported_at datetime,
    PRIMARY KEY(org_id, cluster)
);
