#!/usr/bin/env bash
set -e

mkdir -p /etc/mtail
wget https://github.com/google/mtail/releases/download/v3.0.0-rc53/mtail_3.0.0-rc53_linux_386.tar.gz
tar -xzf mtail_3.0.0-rc53_linux_386.tar.gz
mv mtail /usr/local/bin/mtail
chmod +x /usr/local/bin/mtail
wget https://raw.githubusercontent.com/Bnei-Baruch/gxydb-api/master/misc/nginx.mtail -O /etc/mtail/nginx.mtail

useradd -r exporter

cat <<EOT > /etc/systemd/system/mtail.service
[Unit]
Description=Prometheus mtail
User=exporter
Group=exporter
After=local-fs.target network-online.target network.target
Wants=local-fs.target network-online.target network.target

[Service]
Type=simple
Restart=on-failure
ExecStart=/usr/local/bin/mtail --logs /var/log/nginx/access_gxydb.log  --progs /etc/mtail/

[Install]
WantedBy=multi-user.target
EOT


systemctl daemon-reload
systemctl enable mtail.service
systemctl start mtail.service
