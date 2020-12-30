THINFOIL
========

manages wireguard tunnels between machines, so they all form a starfeld VPC


the vpc network is 172.27.0.0/15 because:

- 129.168.0.0/16    collides with residential
- 10.0.0.0/8        is excessively large and popular because its easy to remember
- 169.254.0.0/16    likely has dangerous policies somewhere, because its assumed physically local
- 100.64.0.0/10     collides with fucked up residential ISPs
- 172.16.0.0/12     collides with docker and some super badly fucked up ISPs, but people are used to that


172.27.0.0/24 is reserved for VPC managment services. nodes allocate from the rest of the 130K addresses.
theoretically customers can also add any other ip range if they really want to.


there's always 3 initial thinfoil servers that manage the state of the whole cluster

 - 172.27.0.10
 - 172.27.0.20
 - 172.27.0.30
 - fd27::10
 - fd27::20
 - fd27::30

a client must contact a random one, and retry a different one on failure

in hyperion, these are orcha nodes and in customer VPCs, these are customer service gateways (VPCG).
both have identical apis.



hint for running on shared switch dedi (e.g. hetzner)
=================================================

Make sure you never respond to arp requests from neighbouring machines.
hetzner policy is to shut down machines that respond to arp requests with unroutable IPs

    sudo sysctl net.ipv4.conf.all.arp_ignore=2


bootstrap
=========

every customer gets a free VPCG if they have at least one machine, there's no need to bootstrap the command nodes.
simply manage the vpc from the cloud web ui.

an orcha cluster for hyperion is a bit difficult to bootstrap due to circular dependency. 
consul won't stabilize without seeing the other nodes, so you can't store the initial keys there

thinfoil must be started on 3 nodes with synchronized config.json .
it should then establish a 3-cluster which lets consul settle. 
before proceeding, thinfoil adds itself to the consul k/v

