ALTER TABLE raffles
ADD COLUMN old_id TEXT;

UPDATE raffles
SET old_id = id::TEXT;

ALTER TABLE raffles
ALTER COLUMN old_id SET NOT NULL;

ALTER TABLE raffles
DROP CONSTRAINT raffles_pkey;

ALTER TABLE raffles
DROP COLUMN id;

ALTER TABLE raffles
RENAME COLUMN old_id TO id;

ALTER TABLE raffles
ADD CONSTRAINT raffles_pkey PRIMARY KEY (id);

DROP SEQUENCE IF EXISTS raffles_id_seq;
