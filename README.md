# setup-etcd-peers-environment
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
        Description=Setup etcd Peers Environment
        Documentation=https://github.com/glerchundi/setup-etcd-peers-environment
        Requires=network-online.target
        After=network-online.target
        [Service]
        Restart=on-failure
        RestartSec=10
        ExecStartPre=-/usr/bin/mkdir -p /opt/bin /etc/sysconfig
        ExecStartPre=/usr/bin/curl -L -o /opt/bin/setup-etcd-peers-environment -z /opt/bin/setup-etcd-peers-environment https://github.com/glerchundi/setup-etcd-peers-environment/releases/download/v0.1.4/setup-etcd-peers-environment
        ExecStartPre=/usr/bin/chmod +x /opt/bin/setup-etcd-peers-environment
        ExecStart=/opt/bin/setup-etcd-peers-environment -o /etc/sysconfig/etcd-peers
    - name: etcd2.service
      command: start
      drop-ins:
        - name: 99-etcd-peers.conf
          content: |
            [Unit]
            Requires=etcd-peers.service
            After=etcd-peers.service
            [Service]
            # Load the other hosts in the etcd leader autoscaling group from file
            EnvironmentFile=/etc/sysconfig/etcd-peers
    - name: fleet.service
      command: start
```
