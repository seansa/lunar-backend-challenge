CREATE TABLE rockets (
    channel VARCHAR(255) NOT NULL,
    rocket_type VARCHAR(255) NOT NULL DEFAULT '',
    mission VARCHAR(255) NOT NULL DEFAULT '',
    mission_changes INT UNSIGNED NOT NULL DEFAULT 0,
    speed BIGINT NOT NULL DEFAULT 0,
    launch_speed BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'unknown',
    exploded_reason TEXT NOT NULL,
    last_message_number BIGINT UNSIGNED NOT NULL DEFAULT 0,
    last_message_time DATETIME(6) NULL,
    launched_at DATETIME(6) NULL,
    exploded_at DATETIME(6) NULL,
    events_applied INT UNSIGNED NOT NULL DEFAULT 0,
    updated_at DATETIME(6) NULL,
    PRIMARY KEY (channel)
) ENGINE = InnoDB;
