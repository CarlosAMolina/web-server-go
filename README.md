# web-server-go

## Introduction

A web server written in go.

## Configuration

### Create certificates

```bash
make certs
```

Update the config.json file with your values.

## Run

After executing `make build`, run:

```bash
./web-server -config $PATH
```

PATH is the path of the config.json file to use.
