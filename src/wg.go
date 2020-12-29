package src


import (
    "golang.zx2c4.com/wireguard/wgctrl"
    "golang.zx2c4.com/wireguard/wgctrl/wgtypes"
    "github.com/vishvananda/netlink"
    "fmt"
    "net"
    "time"
    "log"
)



func WgStart() {

    // bring up link
    link, _ := netlink.LinkByName(Config.Cluster)
    if link != nil {
        netlink.LinkDel(link)
    }
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
        if err != nil { panic(fmt.Errorf("netlink add route %s: %w", route, err)) }
    }
}


func WgPeers(peers []map[string]interface{}) {
    log.Printf("syncing %d peers\n", len(peers));

    psk, err := wgtypes.ParseKey(Config.PresharedKey);
    if err != nil { panic(fmt.Errorf("config.json preshared_key: %w", err)) }


    config := wgtypes.Config {}
    config.ReplacePeers = true;

    for _,peer := range peers {
        keepalive := 20 * time.Second;

        pubkey_s, ok := peer["public_key"].(string);
        if !ok {continue}
        publickey, err := wgtypes.ParseKey(pubkey_s);
        if err != nil {continue}

        endpoint_s, ok := peer["endpoint"].(string);
        if !ok {continue}
        endpoint, err := net.ResolveUDPAddr("udp", endpoint_s);
        if err != nil {continue}


        routes_s, ok := peer["routes"].([]interface{});
        if !ok {continue}
        routes := make([]net.IPNet,0)
        for _,route_ := range routes_s {
            route, ok := route_.(string);
            if !ok {continue}

            _, net , err := net.ParseCIDR(route)
            if err != nil { continue }
            if net == nil { continue }
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

        config.Peers = append(config.Peers, pc);
    }


    wg, err := wgctrl.New();
    if err != nil {
        panic(err)
    }
    defer wg.Close();
    err = wg.ConfigureDevice(Config.Cluster, config);
    if err != nil { panic(err) }

}
