-- 012 already applied on existing databases still rejects theme 'shop'.
-- Fresh installs create the wider check in 012; this migration replaces either form.
ALTER TABLE platform_appearance DROP CONSTRAINT IF EXISTS platform_appearance_theme_check;
ALTER TABLE platform_appearance
    ADD CONSTRAINT platform_appearance_theme_check
    CHECK (theme IN ('classic', 'exhibition', 'shop'));
