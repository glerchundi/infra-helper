Based on: [etcd Clustering in AWS - Configuring a robust etcd cluster in an AWS Auto Scaling Group](http://engineering.monsanto.com/2015/06/12/etcd-clustering/) by [T.J. Corrigan](https://github.com/tj-corrigan)

# infra-helper
Create an environment file with etcd peers based on cloud providers auto scaling facilities.

`cloud-config.yml`
```
#cloud-config
coreos:

  update:
    group: stable
    reboot-strategy: off

  etcd2:
    data-dir: /var/lib/etcd2
    advertise-client-urls: http://$private_ipv4:2379
    initial-advertise-peer-urls: http://$private_ipv4:2380
    listen-client-urls: http://0.0.0.0:2379
    listen-peer-urls: http://$private_ipv4:2380

  units:

    - name: etcd-peers.service
      command: start
      content: |
        [Unit]
        Description=Syncs etcd cluster and deploys a cluster config
        Documentation=https://github.com/glerchundi/infra-helper
        Requires=network-online.target
        After=network-online.target
        [Service]
        Environment=VER=0.1.0
        ExecStartPre=-/usr/bin/mkdir -p /opt/bin
        ExecStartPre=/usr/bin/curl -L -o /opt/bin/infra-helper -z /opt/bin/infra-helper https://github.com/glerchundi/infra-helper/releases/download/v$VER/infra-helper-$VER-linux-amd64
        ExecStartPre=/usr/bin/chmod 0755 /opt/bin/infra-helper
        ExecStart=/opt/bin/infra-helper sync-etcd-peers \
        --out /etc/infra-etcd-initial-cluster.conf
        Restart=on-failure
        RestartSec=10

    - name: etcd2.service
      command: start
      drop-ins:
        - name: 99-etcd-peers.conf
          content: |
            [Unit]
            Requires=etcd-peers.service
            After=etcd-peers.service
            [Service]
            EnvironmentFile=/etc/infra-etcd-initial-cluster.conf

    - name: fleet.service
      command: start
```
