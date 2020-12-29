


there's always 3 servers

169.254.1.2
169.254.1.3
169.254.1.4

a client must contact a random one, and retry a different one on failure

in hyperion, these are orcha nodes
in customer VPCs, these are customer service gateways
both have the same unauthenticated api



bootstrap
=========

every customer gets a free VPCG if they have at least one machine,
so they just add external nodes via the cloud ui

an orcha cluster for hyperion is a bit difficult to bootstrap due to circular dependency
consul won't stabilize without seeing the other nodes, so you can't store the initial keys there

we could just use wg-quick to manually start the cluster by hand,
and later have it updated via thinfoil when consul settled.
care must be taken to add all the keys to consul BEFORE starting thinfoil,
otherwise it will rip the cluster apart again.




key expiry
===========


originally i wanted to use servives to expire keys, but that requires every peer to have a consul agent running.
it also means i can't create them by hand.

so K/V it is, but there's a very slow moving reaper that deletes expired entries,
and updates the ones that have been seen.
this scales poorly, but it'll work fine for a couple hundred keys if the expiry is really slow.
