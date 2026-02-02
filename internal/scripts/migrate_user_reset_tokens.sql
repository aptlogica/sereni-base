-- Temporary migration script for user_reset_tokens table
-- Removes 'expiry' column if it exists and adds 'issued_at' column if not present

DO $$
BEGIN
    -- Check if table exists
    IF EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_name = 'user_reset_tokens'
    ) THEN
        -- Remove 'expiry' column if it exists
        IF EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'user_reset_tokens' AND column_name = 'expiry'
        ) THEN
            ALTER TABLE user_reset_tokens DROP COLUMN expiry;
        END IF;

        -- Add 'issued_at' column if it does not exist
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'user_reset_tokens' AND column_name = 'issued_at'
        ) THEN
            ALTER TABLE user_reset_tokens ADD COLUMN issued_at varchar NOT NULL DEFAULT (extract(epoch from now())::bigint::text);
        END IF;
    END IF;
END $$;
