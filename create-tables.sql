DROP TABLE IF EXISTS labels;
DROP TABLE IF EXISTS artists;
DROP TABLE IF EXISTS albums;
DROP TABLE IF EXISTS label;


CREATE TABLE labels (
    id INT AUTO_INCREMENT NOT NULL,
    name VARCHAR(128) NOT NULL,
    country VARCHAR(128) NOT NULL,
    PRIMARY KEY (`id`)
);

CREATE TABLE albums (
    id INT AUTO_INCREMENT NOT NULL,
    title VARCHAR(128) NOT NULL,
    price DECIMAL(5,2) NOT NULL,
    label_id INT,
    PRIMARY KEY (`id`),
    FOREIGN KEY (label_id) REFERENCES labels(id)
);

CREATE TABLE artists (
     id INT AUTO_INCREMENT NOT NULL,
     name VARCHAR(255) NOT NULL,
     album_id INT,
     PRIMARY KEY (`id`),
     FOREIGN KEY (album_id) REFERENCES albums(id)
);

INSERT INTO labels
(name, country)
VALUES
    ('S1MPLE', 'USA'),
    ('VARCHARTED','GREAT_BRITAIN'),
    ('SANANDREAS', 'BRASIL'),
    ('KARPATY', 'UKRAINE');

INSERT INTO albums
(title, price, label_id)
VALUES
    ('Blue Train',  56.99, 1);

INSERT INTO artists
(name, album_id)
VALUES
    ('Billie Eilish', 1),
    ('Nil Diamond', 1);