### Создаем локальный ключ CA
```openssl genrsa -out ca.key 2048```

### Создаем корневой сертификат CA
```openssl req -new -x509 -days 365 -key ca.key -subj "/C=RU/ST=exampleState/L=exampleLocality/O=exampleOrg, Inc./CN=exampleOrg Root CA" -out ca.crt```

### Создаем .key и .csr сервера
```openssl req -newkey rsa:2048 -nodes -keyout server.key -subj "/C=RU/ST=exampleState/L=exampleLocality/O=exampleOrg, Inc./CN=localhost" -out server.csr```

### Создаем сертификат сервера с subjectAltName полученным из файла sans.txt
```openssl x509 -req -extfile sans.txt -days 365 -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt```

### Создаем ключ и сертификат клиента
```openssl req -newkey rsa:2048 -nodes -keyout client.key -subj "/C=RU/ST=exampleState/L=exampleLocality/O=exampleOrg, Inc./CN=localhost" -out client.csr```
```openssl x509 -req -extfile sans.txt -days 365 -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt```
