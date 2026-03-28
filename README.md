# web-server-go

## Introduction

A web server written in go.

## Configuration

### Create certificates for local testing

```bash
make certs
```

Update the config.json file with your values.

### Binary

```bash
make build
sudo cp web-server /usr/local/bin/web-server
sudo mkdir -p /etc/web-server
sudo cp config.json /etc/web-server/config.json  # update with your values
# Certificates
cp fullchain1.pem /etc/web-server/
cp privkey1.pem /etc/web-server/
sudo chown root:www-data /etc/web-server/privkey1.pem
sudo chmod 640 /etc/web-server/privkey1.pem
```

### Logs

```bash
sudo mkdir /var/log/cmoli.es
sudo chown root:www-data /var/log/cmoli.es
sudo chmod 775 /var/log/cmoli.es
```

### Service

Let's create a systemd service to manage the binary.

Create `sudo vi /etc/systemd/system/web-server.service`:

```ini
[Unit]
Description=web-server-go
# Start after basic networking is up
After=network.target

[Service]
ExecStart=/usr/local/bin/web-server -config /etc/web-server/config.json
# Set working directory (useful if the app uses relative paths)
WorkingDirectory=/etc/web-server
# Restart the service only if it crashes (non-zero exit)
Restart=on-failure
# Wait 5 seconds before restarting (prevents rapid crash loops)
RestartSec=5
# Run the process as a non-root user for security
User=www-data

[Install]
# If this service is enabled (sudo systemctl enable web-server), attach it to multi-user.target
# multi-user.target = the system is fully booted in “normal server mode” (no GUI)
WantedBy=multi-user.target
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
