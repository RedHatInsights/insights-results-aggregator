CREATE TABLE recommendation_0 (
       CHECK (org_id >=0 AND org_id < 10000000)
) INHERITS (recommendation_master);

CREATE TABLE recommendation_1 (
       CHECK (org_id >=10000000 AND org_id < 20000000)
) INHERITS (recommendation_master);

CREATE TABLE recommendation_2 (
       CHECK (org_id >=20000000 AND org_id < 30000000)
) INHERITS (recommendation_master);

CREATE TABLE recommendation_3 (
       CHECK (org_id >=30000000 AND org_id < 40000000)
) INHERITS (recommendation_master);

CREATE TABLE recommendation_4 (
       CHECK (org_id >=40000000)
) INHERITS (recommendation_master);
