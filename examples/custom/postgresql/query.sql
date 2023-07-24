-- template: named_author
CREATE VIEW named_author AS
	SELECT id, lower(name) as fullname, coalese(bio, 'none') as bio_default, 1 as one
	FROM authors;

-- name: ListAuthors :many
SELECT * FROM named_author
ORDER BY fullname;
