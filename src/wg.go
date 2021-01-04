package src


import (
    "golang.zx2c4.com/wireguard/wgctrl"
    "golang.zx2c4.com/wireguard/wgctrl/wgtypes"
    "github.com/vishvananda/netlink"
    "fmt"
    "net"
    "time"
    "log"
    "strings"
)



func WgStart() {

    // bring up link
    link, _ := netlink.LinkByName(Config.Cluster)
    if link == nil {
        wirelink := &netlink.GenericLink{
            LinkAttrs: netlink.LinkAttrs{
                Name: Config.Cluster,
            },
            LinkType: "wireguard",
        };
        err := netlink.LinkAdd(wirelink)
        if err != nil {
            panic(err);
        }
    }

    // bring up wg
    wg, err := wgctrl.New();
    if err != nil {
        panic(err)
    }
    defer wg.Close();


    config := wgtypes.Config {}

    pkey , err := wgtypes.ParseKey(Config.PrivateKey);
    if err != nil { panic(fmt.Errorf("config.json private_key: %w", err)) }
    config.PrivateKey = &pkey;

    endpoint, err := net.ResolveUDPAddr("udp", Config.Endpoint);
    if err != nil { panic(fmt.Errorf("config.json endpoint: %w", err)) }

    config.ListenPort = &endpoint.Port;


    err = wg.ConfigureDevice(Config.Cluster, config);
    if err != nil { panic(err) }

    link, _ = netlink.LinkByName(Config.Cluster)
    for _,route := range Config.Routes {
        addr, err := netlink.ParseAddr(route)
        if err != nil { panic(fmt.Errorf("config.json routes: %w", err)) }
        err = netlink.AddrAdd(link, addr)
        if err != nil {
            log.Println(fmt.Errorf("netlink add route %s: %w", route, err))
        }
    }

	err = netlink.LinkSetUp(link);
    if err != nil {
		panic(fmt.Errorf("link up: %w", err));
	}

    for _, network := range Config.Networks {
        _, net , err := net.ParseCIDR(network)
        if err != nil { continue }
        if net == nil { continue }
        netlink.RouteAdd(&netlink.Route{
            Dst:        net,
            LinkIndex:  link.Attrs().Index,
        })
    }

    pm := make([]map[string]interface{}, 0);
    for _, peer := range Config.BootstrapPeers {
        routes := make([]interface{},0);
        for _,r := range peer.Routes {
            routes = append(routes, r);
        }
        pm = append(pm, map[string]interface{}{
            "public_key":   peer.PublicKey,
            "endpoint":     peer.Endpoint,
            "routes":       routes,
        });
    }
    WgPeers(pm);
}


func WgPeers(peers []map[string]interface{}) {
    log.Printf("syncing %d peers\n", len(peers));

    psk, err := wgtypes.ParseKey(Config.PresharedKey);
    if err != nil { panic(fmt.Errorf("config.json preshared_key: %w", err)) }

    nupeers:= make(map[wgtypes.Key]wgtypes.PeerConfig,0);

    for _,peer := range peers {
        keepalive := 20 * time.Second;

        pubkey_s, ok := peer["public_key"].(string);
        if !ok {
            log.Println("skipping, no public_key");
            continue
        }
        publickey, err := wgtypes.ParseKey(pubkey_s);
        if err != nil {
            log.Println("skipping, public_key:", err);
            continue
        }

        var endpoint *net.UDPAddr = nil;
        if endpoint_s, ok := peer["endpoint"].(string); ok {
            endpoint_s = strings.TrimSpace(endpoint_s);
            if endpoint_s != "" {
                endpoint, err = net.ResolveUDPAddr("udp", endpoint_s);
                if err != nil {
                    log.Println("cant parse endpoint ", endpoint_s, err);
                }
            }
        }

        routes_s, ok := peer["routes"].([]interface{});
        if !ok {
            log.Println("skipping, no routes");
            continue
        }
        routes := make([]net.IPNet,0)
        for _,route_ := range routes_s {
            route, ok := route_.(string);
            if !ok {
                log.Println("skipping route, not string");
                continue
            }

            _, net , err := net.ParseCIDR(route)
            if err != nil {
                log.Println("skipping route:", err);
                continue
            }
            if net == nil {
                log.Println("skipping route: no net");
                continue
            }
            routes = append(routes, *net);


        }

        pc := wgtypes.PeerConfig{
            PersistentKeepaliveInterval: &keepalive,
            ReplaceAllowedIPs:  true,
            PresharedKey:       &psk,
            PublicKey:          publickey,
            Endpoint:           endpoint,
            AllowedIPs:         routes,
        }

        nupeers[publickey] = pc;

        /*
        this is a bad idea. user might accidently contact their default GW
        by trying a bunch of ips in the vpc.
        Instead we use Config.Networks to route the entire VPC,
        so contacting a dead ip gets stopped by wg

        link, _ := netlink.LinkByName(Config.Cluster)
        for _,route_ := range routes_s {
            route, ok := route_.(string);
            if !ok {continue}

            _, net , err := net.ParseCIDR(route)
            if err != nil { continue }
            if net == nil { continue }
            netlink.RouteAdd(&netlink.Route{
                Dst:        net,
                LinkIndex:  link.Attrs().Index,
            })
        }
        */

    }


    wg, err := wgctrl.New();
    if err != nil {
        panic(err)
    }
    defer wg.Close();

    existing_device, err := wg.Device(Config.Cluster)
    if err != nil {
        panic(err)
    }

    existing_peers := make(map[wgtypes.Key]*wgtypes.Peer,0)
    for _,v := range existing_device.Peers {
        existing_peers[v.PublicKey] = &v;
    }

    nuconfig := wgtypes.Config{};

    for k,_ := range existing_peers {
        if nu, ok := nupeers[k]; ok {
            // TODO need to check for differences?
            nu.UpdateOnly = true;
            nuconfig.Peers = append(nuconfig.Peers, nu);

            log.Println("update ", k.String());
            delete (nupeers,k);
        } else {
            //remove peers that are no longer in the new config
            nuconfig.Peers = append(nuconfig.Peers, wgtypes.PeerConfig {
                Remove:     true,
                PublicKey:  k,
            });
            log.Println("remove ", k.String());
        }
    }

    // add the rest that is not yet there
    for k,v := range nupeers {
        if k == existing_device.PublicKey {
            continue;
        }
        log.Println("add ", k.String());
        nuconfig.Peers = append(nuconfig.Peers, v);
    }


    err = wg.ConfigureDevice(Config.Cluster, nuconfig);
    if err != nil { panic(err) }
}
