DROP INDEX IF EXISTS admins_email_lower_unique;

ALTER TABLE admins
    ADD CONSTRAINT admins_email_unique UNIQUE (email);