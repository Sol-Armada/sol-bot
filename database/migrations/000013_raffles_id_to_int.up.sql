CREATE SEQUENCE IF NOT EXISTS raffles_id_seq;

ALTER TABLE raffles
ADD COLUMN new_id INTEGER;

WITH ordered AS (
	SELECT
		id,
		ROW_NUMBER() OVER (ORDER BY created_at, id)::INTEGER AS new_id
	FROM raffles
)
UPDATE raffles r
SET new_id = o.new_id
FROM ordered o
WHERE r.id = o.id;

ALTER TABLE raffles
ALTER COLUMN new_id SET NOT NULL;

ALTER TABLE raffles
DROP CONSTRAINT raffles_pkey;

ALTER TABLE raffles
DROP COLUMN id;

ALTER TABLE raffles
RENAME COLUMN new_id TO id;

ALTER TABLE raffles
ADD CONSTRAINT raffles_pkey PRIMARY KEY (id);

ALTER SEQUENCE raffles_id_seq OWNED BY raffles.id;

ALTER TABLE raffles
ALTER COLUMN id SET DEFAULT nextval('raffles_id_seq');

SELECT setval(
	'raffles_id_seq',
	COALESCE((SELECT MAX(id) FROM raffles), 1),
	EXISTS (SELECT 1 FROM raffles)
);
