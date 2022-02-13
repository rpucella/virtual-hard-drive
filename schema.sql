CREATE TABLE drives (
  id integer primary key,
  name text,
  description text,
  host text,
  address text
);

CREATE TABLE directories (
  id integer primary key,
  driveId integer,
  name text,
  parentId integer
);

CREATE TABLE files (
  id integer primary key,
  driveId integer,
  name text,
  directoryId integer,
  uuid text,
  created int,
  updated int,
  metadata text
);
