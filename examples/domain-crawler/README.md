# worker-pool

This demonstrational application shows how to implement a pool of workers to execute different tasks simultaneously.

## Overview

The application takes a list of domains (as a file with one domain per
line) from the standard input and outputs the average response time and the average data size of the download of the
index pages across all of the input domains.
Since the list of domains can potentially be very large (or streamed) and unknown, we want to do this in a controllable way.
Doing it serially is too slow and doing everything at once is not scalable.
That's why it uses a generic work pool to control the processing described above.

### Command line flags
```
   -t int
      HTTP timeout. (default 10)
   -w int
      Number of workers. (default 10)
```

### Run unit tests

```
go test -race ./...
```

### Build the binary

```
go build
```

### Run manual tests

   ```
   $ ./domain-crawler -t 10 -w 30 < top111.txt 
   processing started with 30 workers
   success: https://github.com, size 309937, duration 280.099034ms
   success: https://google.com, size 15075, duration 439.817687ms
   ...   
   downloaded 95 files, average 203507 bytes, 1.158814315s
   ```
