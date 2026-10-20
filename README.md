# Docker Restart Policy Guard

Docker authorization plugin that prevents containers from being created with restart policies in HPC/batch scheduling environments.

## Problem

In resource-managed environments (SLURM, Kubernetes, etc.), containers with restart policies can survive job cleanup, causing resource leaks when Docker restarts them during the gap between process termination and cgroup destruction.

## Solution

This authorization plugin intercepts container creation requests and rejects any that specify a restart policy other than `no`.

## Installation

### From Release

Download the latest binary from [releases](https://github.com/reboss/docker-restart-policy-guard/releases):

```bash
curl -L https://github.com/reboss/docker-restart-policy-guard/releases/latest/download/docker-restart-policy-guard-linux-amd64 \
  -o /usr/local/bin/docker-restart-policy-guard
chmod +x /usr/local/bin/docker-restart-policy-guard
```

### Systemd Service

Create `/etc/systemd/system/docker-restart-policy-guard.service`:

```ini
[Unit]
Description=Docker Restart Policy Guard Authorization Plugin
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/docker-restart-policy-guard
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
systemctl daemon-reload
systemctl enable docker-restart-policy-guard
systemctl start docker-restart-policy-guard
```

### Docker Configuration

Create plugin manifest at `/etc/docker/plugins/restart-policy-guard.json`:

```json
{
  "Name": "restart-policy-guard",
  "Addr": "unix:///run/docker/plugins/restart-policy-guard.sock"
}
```

Enable plugin in `/etc/docker/daemon.json`:

```json
{
  "authorization-plugins": ["restart-policy-guard"]
}
```

Restart Docker:

```bash
systemctl restart docker
```

## Usage

Once installed, attempts to create containers with restart policies will fail:

```bash
$ docker run --restart=always alpine
Error response from daemon: authorization denied by plugin restart-policy-guard: Container restart policies are not allowed in this cluster (attempted: always)
```

## Building

```bash
go build -o docker-restart-policy-guard
```

## License

MIT
