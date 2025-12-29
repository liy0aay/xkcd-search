CREATE TABLE comics (
    id int PRIMARY KEY,
    url TEXT NOT NULL,
    title TEXT DEFAULT '',
    alt TEXT DEFAULT '',
    words TEXT[]
);