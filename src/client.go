package src;



import (
    "log"
    "time"
    "encoding/json"
    "github.com/gorilla/websocket"
    "math/rand"
    "reflect"
)


func init() {
    rand.Seed(time.Now().Unix())
}


func Client() {

    ConfigLoad();
    WgStart();

    if len(Config.Routes) < 1 {
        panic("no routes in config.json");
    }

    VPCGurls := []string{
        "ws://172.27.0.20/thinfoil/peers",
        "ws://172.27.0.30/thinfoil/peers",
        "ws://172.27.0.40/thinfoil/peers",
    }

    rurl := VPCGurls[rand.Intn(len(VPCGurls))];

    log.Println("connecting to", rurl);


    websocket.DefaultDialer.HandshakeTimeout = 2 * time.Second;
    c, _, err := websocket.DefaultDialer.Dial(rurl, nil);
    if err != nil {
        log.Fatal("dial:", err)
    }
    defer c.Close()

    for {
        _, message, err := c.ReadMessage()
        if err != nil { panic(err) }

        log.Println(string(message));

        m := make(map[string]interface{});
        err = json.Unmarshal(message, &m);
        if err != nil { panic(err) }

        if l , ok := m["peers"].([]interface{}); ok {
            wgpeers        := make([]map[string]interface{}, 0);
            boostrap_peers := make([]BootstrapPeer, 0);
            for _,v := range l {
                m := v.(map[string]interface{});
                wgpeers = append(wgpeers, m);


                routes := make([]string, 0)
                for _,v := range m["routes"].([]interface{}) {
                    routes = append(routes, v.(string));
                }
                ep, _ := m["endpoint"].(string)
                boostrap_peers = append(boostrap_peers, BootstrapPeer{
                    Endpoint:       ep,
                    PublicKey:      m["public_key"].(string),
                    Routes:         routes,
                });
            }
            WgPeers(wgpeers);

            if !reflect.DeepEqual(&Config.BootstrapPeers, &boostrap_peers) {
                log.Println("saved current peers to config.json");
                Config.BootstrapPeers = boostrap_peers;
                ConfigStore();
            }
        }
    }
}
