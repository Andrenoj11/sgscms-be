ALTER TABLE admins
    DROP CONSTRAINT IF EXISTS admins_email_unique;

CREATE UNIQUE INDEX admins_email_lower_unique
    ON admins (LOWER(email));