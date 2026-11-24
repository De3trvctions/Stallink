CREATE TABLE platform_appearance (
    singleton boolean PRIMARY KEY CHECK (singleton),
    theme text NOT NULL CHECK (theme IN ('classic', 'exhibition'))
);
