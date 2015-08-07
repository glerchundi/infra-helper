Based on: [etcd Clustering in AWS - Configuring a robust etcd cluster in an AWS Auto Scaling Group](http://engineering.monsanto.com/2015/06/12/etcd-clustering/) by [T.J. Corrigan](https://github.com/tj-corrigan)

# infra-helper
Create an environment file with etcd peers based on cloud providers auto scaling facilities.

Usage:
```
$> ./bin/infra-helper --help
NAME:
   infra-helper - manage etcd cluster based on AWS autoscaling groups

USAGE:
   infra-helper [global options] command [command options] [arguments...]
   
VERSION:
   0.1.1
   
COMMANDS:
   sync-etcd-peers    syncs "etcd" cluster (adds/removes members based on 'autoscale' information)
   list-autoscale-members 
   help, h      Shows a list of commands or help for one command
   
GLOBAL OPTIONS:
   --help, -h   show help
   --version, -v  print the version
```

Usage for **sync-etcd-peers**:
```
$> ./bin/infra-helper sync-etcd-peers --help
NAME:
   sync-etcd-peers - syncs "etcd" cluster (adds/removes members based on 'autoscale' information)

USAGE:
   command sync-etcd-peers [command options] [arguments...]

OPTIONS:
   --out, -o "/etc/sysconfig/etcd-peers"  etcd peers environment file destination
```

Usage for **list-autoscale-members**:
```
$> ./bin/infra-helper list-autoscale-members --help
NAME:
   list-autoscale-members - 

USAGE:
   command list-autoscale-members [command options] [arguments...]

OPTIONS:
   --name, -n               search by name
   --format, -f "{{range .}}{{.Name}}={{.Address}}\n{{end}}"  defines how to format members output
   -c, --chomp              chomp an ending delimiter off template's output
   --out, -o 
```

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
