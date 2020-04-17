USE bastion;

INSERT INTO users(pk, name, last_login) VALUES (1, 'S-1-5-21-2382012410-1563639239-1097593746-5019', current_timestamp());

INSERT INTO target_credentials(pk, target_login, target_password, target_private_key) VALUES (1, 'sshtest', 'sshtest', null);
INSERT INTO target_credentials(pk, target_login, target_password, target_private_key) VALUES (2, 'teltest', 'teltestPassw0rd', null);

INSERT INTO networks(pk, name, endpoint, servicepoint) VALUES (1, 'NT1', 'nt1-proxy.internal.example.com:2200', 'bastion.example.com:2201');
INSERT INTO networks(pk, name, endpoint, servicepoint) VALUES (2, 'NT2', 'nt2-proxy.internal.example.com:2200', 'bastion.example.com:2202');
INSERT INTO networks(pk, name, endpoint, servicepoint) VALUES (3, 'NT3', '', 'localhost:2203');
INSERT INTO networks(pk, name, endpoint, servicepoint) VALUES (4, 'NT4', 'nt4-proxy.internal.example.com:2200', 'bastion.example.com:2204');

INSERT INTO mandates(pk, name, network_id, target_credentials_id) VALUES (1, 'Мандат с паролем на SSH Test Server в NT3', 3, 1);
INSERT INTO mandates(pk, name, network_id, target_credentials_id) VALUES (2, 'Мандат с паролем на Telnet Test Server в NT3', 3, 2);

INSERT INTO users_mandates(user_id, mandate_id) VALUES (1, 1), (1, 2);

INSERT INTO session_templates(pk, name, user_id, target_proto_id, target_host, target_port, mandate_id)
VALUES (1, 'SSH Test server', 1, 1, '10.73.0.2', 22, 1);
INSERT INTO session_templates(pk, name, user_id, target_proto_id, target_host, target_port, mandate_id)
VALUES (2, 'Telnet Test server', 1, 2, '10.73.0.3', 23, 2);
INSERT INTO session_templates(pk, name, user_id, target_proto_id, target_host, target_port, mandate_id, custom_target_network_id, custom_target_login, custom_target_password)
VALUES (3, 'SSH Test server with custom credentials', 1, 1, '10.73.0.2', 22, null, 3, 'sshtest', 'sshtest');
