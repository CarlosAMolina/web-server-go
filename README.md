# web-server-go

## Introduction

A web server written in go.

## Configuration

### Certificates

#### VPS

Read [this tutorial](https://cmoli.es/wiki/content/certbot/certbot.html).

#### Local testing

```bash
make certs
```

If you run the server with `make run`, not additional action is required. But if you will run the server as a service (explained below), you will need:

```bash
cp server.cert /tmp/
cp server.key /tmp/
chmod 604 /tmp/server.key
```

### Binary

```bash
make build
sudo cp web-server /usr/local/bin/web-server
# Configuration
sudo mkdir -p /etc/web-server
sudo cp testdata/config-vps.json /etc/web-server/config.json  # update with your values. To run the VPS as a service locally, copy the config-service-local.json instead of config-vps.json.
```

### Web content

```bash
sudo mkdir /var/www/your-domain.com
sudo chown root:www-data /var/www/your-domain.com
sudo chmod 750 /var/www/your-domain.com
```

For local testing, you can:

```bash
sudo cp -r testdata/content/* /var/www/your-domain.com
```

### Logs

```bash
sudo mkdir /var/log/your-domain.com
sudo chown root:www-data /var/log/your-domain.com
sudo chmod 775 /var/log/your-domain.com
```

### Service

Let's create a systemd service to manage the binary:

```bash
sudo cp testdata/web-server.service /etc/systemd/system/
```

### Redirect port request

We will use iptables.

Redirect port 80 requests to 8080:

```bash
sudo iptables -t nat -A PREROUTING -p tcp --dport 80  -j REDIRECT --to-port 8080
sudo iptables -t nat -A PREROUTING -p tcp --dport 443 -j REDIRECT --to-port 8443
```

Make the rules persistent to not loose them on reboot:

```bash
sudo apt update
sudo apt install iptables-persistent
# During install, it may ask to save current rules, say YES. If not, do it manually:
sudo netfilter-persistent save
```

The rules are saved at `sudo cat sudo /etc/iptables/rules.v4`.

## Run

### Enable and start

```bash
sudo systemctl daemon-reload  # detect new service.
sudo systemctl enable web-server  # auto-start on boot
```

### Change and review status

```bash
sudo systemctl start web-server
sudo systemctl stop web-server
sudo systemctl restart web-server
sudo systemctl status web-server
```

## Logs

To see system logs:

```bash
sudo journalctl -u web-server.service -f
# Filter errors (TODO not verified, is always empty)
sudo journalctl -u web-server.service -p err
```

TODO configure custom log file that rotates using linux tools isntead of go log rotate library: 
```bash
sudo mkdir -p /etc/systemd/journald.conf.d/
# testdata/web-server.conf
# [Journal]
# SystemMaxFileSize=5M
# SystemMaxFiles=5
# Storage=persistent
sudo cp testdata/web-server.conf /etc/systemd/journald.conf.d/
sudo systemctl restart systemd-journald
```
