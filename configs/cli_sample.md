## Запуск bastion-proxy из командной строки
```
./bastion-proxy --log-level debug
--api-cert test/devbench/cert.pem
--api-url https://localhost:1443
--bind-address 0.0.0.0:2203
--network GRT
--oidc-client-id bastion-proxy
--oidc-client-secret <SECRET>
--oidc-issuer https://idp.example.com/
```

## Запуск bastion-server из командной строки
```
./bastion-server --log-level debug
--log-requests
--datastore-dsn bastion:bastion@tcp(10.69.0.2)/bastion
--oidc-allowed-client-id bastion-proxy
--oidc-allowed-client-id bastion-proxy-another
--oidc-client-id bastion-server
--oidc-client-secret <SECRET>
--oidc-issuer https://idp.example.com/
--oidc-redirect-url https://192.168.1.4:1443/auth/callback
--tls-cert-file test/devbench/cert.pem
--tls-key-file test/devbench/key.pem
--web-static web/webroot/static
--web-templates web/templates
```
