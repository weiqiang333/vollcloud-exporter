# vollcloud-exporter
[vollcloud](https://vollcloud.com/) cloud resource monitoring "vollcloud-exporter", Use golang to provide high-performance metrics API

Mertrics api exposing vollcloud server information. [mertrics_example](./docs/mertrics_example)

[中文](README-cn.md)

## usage
Install and Start
```shell
wget https://github.com/weiqiang333/vollcloud-exporter/releases/download/v1.0/vollcloud-exporter-linux-amd64-v1.0.tar.gz
mkdir /usr/local/vollcloud-exporter
tar -zxf vollcloud-exporter-linux-amd64-v1.0.tar.gz -C /usr/local/vollcloud-exporter
chmod +x /usr/local/vollcloud-exporter/vollcloud-exporter
/usr/local/vollcloud-exporter/vollcloud-exporter --config.file /usr/local/vollcloud-exporter/config/vollcloud-exporter.yaml
    # Don't forget to modify your config file /usr/local/vollcloud-exporter/config/vollcloud-exporter.yaml
```

Flags
```
      --address string      The address on which to expose the web interface and generated Prometheus metrics. (default ":9109")
      --configfile string   exporter config file (default "./config/vollcloud-exporter.yaml")
```

### systemd administer service
```
cp /usr/local/vollcloud-exporter/config/vollcloud-exporter.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now vollcloud-exporter
systemctl status vollcloud-exporter
```

### API
```
    http://127.0.0.1:9109/metrics
    http://127.0.0.1:9109/reload  # reload default "config/config.yaml"
```

---
## prometheus
query prometheus.


## grafana
The following [Dashboard template](./docs/granfana.json), can be imported into grafana to get an basic dashboard.

Example:

