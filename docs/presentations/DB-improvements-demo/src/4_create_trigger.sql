CREATE TRIGGER insert_recommendation_trigger
    BEFORE INSERT ON recommendation_master
    FOR EACH ROW EXECUTE PROCEDURE recommendation_insert_trigger();

CREATE OR REPLACE FUNCTION recommendation_insert_trigger()
RETURNS TRIGGER AS $$
BEGIN
    IF ( NEW.org_id >= 0 AND
         NEW.org_id < 10000000 ) THEN
        INSERT INTO recommendation_0 VALUES (NEW.*);
    ELSIF ( NEW.org_id >= 10000000 AND
            NEW.org_id < 20000000 THEN
        INSERT INTO recommendation_1 VALUES (NEW.*);
    ...
    ELSIF ( NEW.org_id >= 20000000 AND
            NEW.org_id < 30000000 THEN
        INSERT INTO recommendation_2 VALUES (NEW.*);
    ELSE
        RAISE EXCEPTION 'org_id out of range.  Fix the measurement_insert_trigger() function!';
    END IF;
    RETURN NULL;
END;
$$
LANGUAGE plpgsql;
