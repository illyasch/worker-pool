# password-bcrypt-service

## Overview

The service uses the bcrypt algorithm to encrypt an incoming password to a hash string.
As bcrypt may be computationally expensive, the service uses the pool of workers to do the job.
If there are too many incoming requests and all workers are busy more than given _BusyTimeout_,
the service returns 429 status Too Many Requests.

- _/bcrypt_ - use the POST method and x-www-form-urlencoded parameter password.
  Returns bcrypt encrypted password.

## How to

### Run unit tests

```
go test -race -v ./...
```

### Build and run the service

```
$ go build .
$ ./password-bcrypt-service
BCRYPT: 2022/12/09 17:07:25 startup config --build=
--desc=Copyright Ilya Scheblanov
--api-host=0.0.0.0:3000
--num-workers=10
--shutdown-timeout=20s
--busy-timeout=100ms
BCRYPT: 2022/12/09 17:07:25 starting service
BCRYPT: 2022/12/09 17:07:25 startup status initializing API support
BCRYPT: 2022/12/09 17:07:25 startup status srv router started host 0.0.0.0:3000
BCRYPT: 2022/12/09 17:09:25 bcrypt password SUCCESS
BCRYPT: 2022/12/09 17:09:25 bcrypt statusCode 200 method POST path /bcrypt remoteaddr 127.0.0.1:37604
^CBCRYPT: 2022/12/09 17:10:56 shutdown status shutdown started signal interrupt
BCRYPT: 2022/12/09 17:10:56 shutdown status shutdown complete signal interrupt
BCRYPT: 2022/12/09 17:10:56 shutdown complete
```

### Run manual tests

Bcrypt a password
  ```
  $ curl -i --data-urlencode "password=ldrrjlkjkrlkrfj335535" http://localhost:3000/bcrypt
   
  HTTP/1.1 200 OK
  Content-Type: application/json
  Date: Fri, 09 Dec 2022 16:09:25 GMT
  Content-Length: 71
  
  {"hash":"$2a$10$eh4WDYPN7td.uuZtRYcOmO8eP6UyyJUSm6UljxM7YmleVZmbMx77e"}
  ```