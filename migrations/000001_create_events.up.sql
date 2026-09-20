CREATE TABLE events (
    channel VARCHAR(255) NOT NULL,
    message_number BIGINT UNSIGNED NOT NULL,
    message_type VARCHAR(64) NOT NULL,
    message_time DATETIME(6) NOT NULL,
    payload JSON NOT NULL,
    PRIMARY KEY (channel, message_number)
) ENGINE = InnoDB;
