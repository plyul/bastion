-- Если надо жоско:
-- REVOKE ALL PRIVILEGES ON bastion.* FROM 'bastion'@'%';
-- DROP USER 'bastion'@'%';
-- DROP DATABASE bastion;

CREATE DATABASE bastion
    CHARACTER SET 'utf8mb4'
    COLLATE 'utf8mb4_general_ci';
USE bastion;

CREATE USER 'bastion'@'%' IDENTIFIED BY 'bastion';
GRANT ALL PRIVILEGES ON bastion.* TO 'bastion'@'%';

--
-- Таблица содержит список авторизованных пользователей
--
CREATE TABLE users (
    pk INT UNSIGNED NOT NULL AUTO_INCREMENT,
    name CHAR(128) NOT NULL,
    last_login TIMESTAMP,
    PRIMARY KEY (pk),
    KEY `users_name_index` (`name`)
) ENGINE INNODB;

--
-- Таблица содержит поддерживаемые протоколы доступа
--
CREATE TABLE protocols (
    pk INT UNSIGNED NOT NULL AUTO_INCREMENT,
    name CHAR(20) NOT NULL,
    default_port INT UNSIGNED NOT NULL,
    PRIMARY KEY (pk)
) ENGINE INNODB;
INSERT INTO protocols(pk, name, default_port) VALUES (1, 'SSH', 22), (2, 'Telnet', 23);

--
-- Таблица содержит учётные данные для доступа
--
CREATE TABLE target_credentials (
    pk INT UNSIGNED NOT NULL AUTO_INCREMENT,
    target_login CHAR(128) NOT NULL,
    target_password CHAR(128),
    target_private_key VARCHAR(8192),
    PRIMARY KEY (pk)
) ENGINE INNODB;

--
-- Таблица содержит данные о сетях и точках доступа к ним извне (на Ingress)
--
CREATE TABLE networks (
    pk INT UNSIGNED NOT NULL AUTO_INCREMENT,
    name CHAR(128) NOT NULL,
    endpoint CHAR(128) NOT NULL,
    servicepoint CHAR(128) NOT NULL,
    PRIMARY KEY (pk)
) ENGINE INNODB;

--
-- Таблица содержит мандаты доступа к устройствам
--
CREATE TABLE mandates (
    pk INT UNSIGNED NOT NULL AUTO_INCREMENT,
    name CHAR(128) NOT NULL,
    network_id INT UNSIGNED NOT NULL,
    target_credentials_id INT UNSIGNED NOT NULL,
    PRIMARY KEY (pk)
) ENGINE INNODB;
ALTER TABLE mandates ADD CONSTRAINT mandates_target_credentials_fk
    FOREIGN KEY (target_credentials_id) REFERENCES target_credentials(pk)
        ON UPDATE RESTRICT ON DELETE RESTRICT;
ALTER TABLE mandates ADD CONSTRAINT mandates_networks_fk
    FOREIGN KEY (network_id) REFERENCES networks(pk)
        ON UPDATE RESTRICT ON DELETE RESTRICT;

--
-- Таблица содержит отношение пользователя и доступных ему мандатов
--
CREATE TABLE users_mandates (
    user_id INT UNSIGNED NOT NULL,
    mandate_id INT UNSIGNED NOT NULL
) ENGINE INNODB;
ALTER TABLE users_mandates ADD CONSTRAINT users_mandates_users_pk FOREIGN KEY (user_id) REFERENCES users(pk)
    ON UPDATE RESTRICT ON DELETE RESTRICT;
ALTER TABLE users_mandates ADD CONSTRAINT users_mandates_mandates_pk FOREIGN KEY (mandate_id) REFERENCES mandates(pk)
    ON UPDATE RESTRICT ON DELETE RESTRICT;

--
-- Таблица содержит активные пользовательские сессии
--
CREATE TABLE sessions (
    pk INT UNSIGNED NOT NULL AUTO_INCREMENT,
    token CHAR(128) NOT NULL,
    origin_ip CHAR(40) NOT NULL,
    user_id INT UNSIGNED NOT NULL,
    target_proto_id INT UNSIGNED NOT NULL,
    target_host CHAR(128) NOT NULL,
    target_port INT UNSIGNED NOT NULL,
    mandate_id INT UNSIGNED,
    custom_target_network_id INT UNSIGNED,
    custom_target_login CHAR(128),
    custom_target_password CHAR(128),
    custom_target_private_key VARCHAR(8192),
    created_at timestamp NOT NULL DEFAULT current_timestamp(),
    PRIMARY KEY (pk),
    UNIQUE KEY `sessions_token_uindex` (`token`),
    KEY `sessions_users_fk` (`user_id`),
    KEY `sessions_protocols_fk` (`target_proto_id`),
    KEY `sessions_mandates_fk` (`mandate_id`),
    KEY `session_networks_fk` (`custom_target_network_id`),
    CONSTRAINT `sessions_users_fk` FOREIGN KEY (`user_id`) REFERENCES `users` (`pk`),
    CONSTRAINT `sessions_protocols_fk` FOREIGN KEY (`target_proto_id`) REFERENCES `protocols` (`pk`),
    CONSTRAINT `sessions_mandates_fk` FOREIGN KEY (`mandate_id`) REFERENCES `mandates` (`pk`),
    CONSTRAINT `session_networks_fk` FOREIGN KEY (`custom_target_network_id`) REFERENCES `networks` (`pk`)
) ENGINE INNODB;

--
-- Таблица содержит шаблоны сессий, созданные пользователями
--
CREATE TABLE session_templates (
    pk INT UNSIGNED NOT NULL AUTO_INCREMENT,
    name CHAR(128) NOT NULL,
    user_id INT UNSIGNED NOT NULL,
    target_proto_id INT UNSIGNED NOT NULL,
    target_host CHAR(128) NOT NULL,
    target_port INT UNSIGNED NOT NULL,
    mandate_id INT UNSIGNED,
    custom_target_network_id INT UNSIGNED,
    custom_target_login CHAR(128),
    custom_target_password CHAR(128),
    custom_target_private_key VARCHAR(8192),
    PRIMARY KEY (pk),
    KEY `session_templates_users_fk` (`user_id`),
    KEY `session_templates_protocols_fk` (`target_proto_id`),
    KEY `session_templates_mandates_fk` (`mandate_id`),
    KEY `session_templates_networks_fk` (`custom_target_network_id`),
    CONSTRAINT `session_templates_users_fk` FOREIGN KEY (`user_id`) REFERENCES `users` (`pk`),
    CONSTRAINT `session_templates_protocols_fk` FOREIGN KEY (`target_proto_id`) REFERENCES `protocols` (`pk`),
    CONSTRAINT `session_templates_mandates_fk` FOREIGN KEY (`mandate_id`) REFERENCES `mandates` (`pk`),
    CONSTRAINT `session_templates_networks_fk` FOREIGN KEY (`custom_target_network_id`) REFERENCES `networks` (`pk`)
) ENGINE INNODB;
