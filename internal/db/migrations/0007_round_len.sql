-- Round length is a parent setting on the profile, so the child never sees
-- a question-count picker. 5, 10, or 20 math questions; spelling uses 5
-- words up to 10 and 10 words at 20.
ALTER TABLE children ADD COLUMN round_len INTEGER NOT NULL DEFAULT 10 CHECK (round_len IN (5, 10, 20));
