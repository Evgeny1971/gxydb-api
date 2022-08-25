#!/usr/bin/env bash
# run misc/sync_config.sh from project root
# make sure you have generated the config from janus-config command with and up to date DB
# before syncing to gateways
set -e
set -x

# backup first
scp -rq root@gxy1.kab.sh:/usr/janusgxy/etc/janus/ misc/janus/gxy1
scp -rq root@gxy2.kab.sh:/usr/janusgxy/etc/janus/ misc/janus/gxy2
scp -rq root@gxy3.kab.sh:/usr/janusgxy/etc/janus/ misc/janus/gxy3
scp -rq root@gxy4.kab.sh:/usr/janusgxy/etc/janus/ misc/janus/gxy4
scp -rq root@gxy5.kab.sh:/usr/janusgxy/etc/janus/ misc/janus/gxy5
scp -rq root@gxy6.kab.sh:/usr/janusgxy/etc/janus/ misc/janus/gxy6
scp -rq root@gxy7.kab.sh:/usr/janusgxy/etc/janus/ misc/janus/gxy7
scp -rq root@gxy8.kab.sh:/usr/janusgxy/etc/janus/ misc/janus/gxy8
scp -rq root@gxy9.kab.sh:/usr/janusgxy/etc/janus/ misc/janus/gxy9
scp -rq root@gxy10.kab.sh:/usr/janusgxy/etc/janus/ misc/janus/gxy10
scp -rq root@gxy11.kab.sh:/usr/janusgxy/etc/janus/ misc/janus/gxy11
scp -rq root@gxy12.kab.sh:/usr/janusgxy/etc/janus/ misc/janus/gxy12
#scp -rq root@gxy13.kab.sh:/usr/janusgxy/etc/janus/ misc/janus/gxy13
#scp -rq root@gxy14.kab.sh:/usr/janusgxy/etc/janus/ misc/janus/gxy14

# copy videoroom config
scp janus.plugin.videoroom.jcfg root@gxy1.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.videoroom.jcfg
scp janus.plugin.videoroom.jcfg root@gxy2.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.videoroom.jcfg
scp janus.plugin.videoroom.jcfg root@gxy3.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.videoroom.jcfg
scp janus.plugin.videoroom.jcfg root@gxy4.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.videoroom.jcfg
scp janus.plugin.videoroom.jcfg root@gxy5.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.videoroom.jcfg
scp janus.plugin.videoroom.jcfg root@gxy6.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.videoroom.jcfg
scp janus.plugin.videoroom.jcfg root@gxy7.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.videoroom.jcfg
scp janus.plugin.videoroom.jcfg root@gxy8.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.videoroom.jcfg
scp janus.plugin.videoroom.jcfg root@gxy9.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.videoroom.jcfg
scp janus.plugin.videoroom.jcfg root@gxy10.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.videoroom.jcfg
scp janus.plugin.videoroom.jcfg root@gxy11.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.videoroom.jcfg
scp janus.plugin.videoroom.jcfg root@gxy12.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.videoroom.jcfg
#scp janus.plugin.videoroom.jcfg root@gxy13.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.videoroom.jcfg
#scp janus.plugin.videoroom.jcfg root@gxy14.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.videoroom.jcfg

# copy textroom config
scp janus.plugin.textroom.jcfg root@gxy1.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.textroom.jcfg
scp janus.plugin.textroom.jcfg root@gxy2.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.textroom.jcfg
scp janus.plugin.textroom.jcfg root@gxy3.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.textroom.jcfg
scp janus.plugin.textroom.jcfg root@gxy4.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.textroom.jcfg
scp janus.plugin.textroom.jcfg root@gxy5.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.textroom.jcfg
scp janus.plugin.textroom.jcfg root@gxy6.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.textroom.jcfg
scp janus.plugin.textroom.jcfg root@gxy7.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.textroom.jcfg
scp janus.plugin.textroom.jcfg root@gxy8.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.textroom.jcfg
scp janus.plugin.textroom.jcfg root@gxy9.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.textroom.jcfg
scp janus.plugin.textroom.jcfg root@gxy10.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.textroom.jcfg
scp janus.plugin.textroom.jcfg root@gxy11.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.textroom.jcfg
scp janus.plugin.textroom.jcfg root@gxy12.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.textroom.jcfg
#scp janus.plugin.textroom.jcfg root@gxy13.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.textroom.jcfg
#scp janus.plugin.textroom.jcfg root@gxy14.kab.sh:/usr/janusgxy/etc/janus/janus.plugin.textroom.jcfg

ssh root@gxy1.kab.sh "systemctl restart janusgxy"
ssh root@gxy2.kab.sh "systemctl restart janusgxy"
ssh root@gxy3.kab.sh "systemctl restart janusgxy"
ssh root@gxy4.kab.sh "systemctl restart janusgxy"
ssh root@gxy5.kab.sh "systemctl restart janusgxy"
ssh root@gxy6.kab.sh "systemctl restart janusgxy"
ssh root@gxy7.kab.sh "systemctl restart janusgxy"
ssh root@gxy8.kab.sh "systemctl restart janusgxy"
ssh root@gxy9.kab.sh "systemctl restart janusgxy"
ssh root@gxy10.kab.sh "systemctl restart janusgxy"
ssh root@gxy11.kab.sh "systemctl restart janusgxy"
ssh root@gxy12.kab.sh "systemctl restart janusgxy"
#ssh root@gxy13.kab.sh "systemctl restart janusgxy"
#ssh root@gxy14.kab.sh "systemctl restart janusgxy"