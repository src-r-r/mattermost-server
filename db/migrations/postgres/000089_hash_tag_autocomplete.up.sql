CREATE TABLE IF NOT EXISTS hash_tag (
    val VARCHAR(1024),
    post_id varchar(26),
    UNIQUE (val, post_id)
)