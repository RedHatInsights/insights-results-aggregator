CREATE TABLE recommendation_0
PARTITION OF recommendation_master
FOR VALUES FROM (0) TO (10000000);

CREATE TABLE recommendation_1
PARTITION OF recommendation_master
FOR VALUES FROM (10000000) TO (20000000);
