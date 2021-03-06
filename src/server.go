package src;



import (
    "github.com/gin-gonic/gin"
    consulapi   "github.com/hashicorp/consul/api"
    "log"
    "time"
    "net/http"
    "encoding/json"
    "github.com/gorilla/websocket"
    "context"
    "strings"
    "net/url"
)


func Server() {

    ConfigLoad();
    WgStart();

    consul, err := consulapi.NewClient(consulapi.DefaultConfig())
    if err != nil {
        panic(err)
    }

    if len(Config.Routes) < 1 {
        panic("no routes in config.json");
    }

    address := strings.Split(Config.Routes[0], "/")[0];

    for  {
        // register our node
        err = consul.Agent().ServiceRegister(&consulapi.AgentServiceRegistration{
            Address: address,
            Port:   80,
            Name: "thinfoil",
            Checks: consulapi.AgentServiceChecks{
                &consulapi.AgentServiceCheck{
                    Name:       "thinfoil keepalive",
                    CheckID:    "thinfoil:keepalive",
                    TTL:        "30s",
                    //DeregisterCriticalServiceAfter: "120s",
                },
                &consulapi.AgentServiceCheck{
                    Name:       "thinfoil http API",
                    CheckID:    "thinfoil:api",
                    Interval:   "10s",
                    HTTP:       "http://" + address + "/health",
                    Method:     "GET",
                    Timeout:    "1s",
                    //DeregisterCriticalServiceAfter: "120s",
                },
            },
            Meta:  map[string]string{
                "public_key":       Config.publicKey,
                "endpoint":         Config.Endpoint,
                "routes":           strings.Join(Config.Routes, ","),
            },
        });
        if err != nil {
            log.Println(err)
            log.Println("retry in 5s")
            time.Sleep(5 * time.Second);
        } else {
            break;
        }
    }


    // make sure our own system is in k/v

    for {

        val, err := json.Marshal(gin.H{
            "public_key":   Config.publicKey,
            "endpoint":     Config.Endpoint,
            "routes":       Config.Routes,
        });
        if err != nil { panic(err) }

        nodename , err := consul.Agent().NodeName();
        if err != nil { panic(err) }

        _, err = consul.KV().Put(&consulapi.KVPair{
            Key:    "thinfoil/" + url.PathEscape(Config.Cluster) + "/peers/" + url.PathEscape(nodename),
            Value:  val,
        }, nil);

        if err != nil {
            log.Println(err)
            log.Println("retry in 5s")
            time.Sleep(5 * time.Second);
        } else {
            break;
        }
    }


    log.Println("going live in 5");
    time.Sleep(time.Second);
    log.Println("going live in 4");
    time.Sleep(time.Second);
    log.Println("going live in 3");
    time.Sleep(time.Second);
    log.Println("going live in 2");
    time.Sleep(time.Second);
    log.Println("going live in 1");
    time.Sleep(time.Second);

    go func() {
        var index uint64 = 0;
        for {
            err := consul.Agent().UpdateTTL("thinfoil:keepalive", "", "passing");
            if err != nil {
                panic(err);
            }

            kvpairs, meta, err := consul.KV().List("thinfoil/" + url.PathEscape(Config.Cluster) + "/peers/", &consulapi.QueryOptions{
                WaitIndex:  index,
                WaitTime:   20 * time.Second,
            });
            if err != nil { panic(err) }

            index = meta.LastIndex

            peers := make([]map[string]interface{}, 0);
            for _,v := range kvpairs {
                val := make(map[string]interface{});
                err = json.Unmarshal(v.Value, &val);
                if err != nil { continue }
                peers = append(peers, val);
            }
            WgPeers(peers);

        }
    }();


    router := gin.Default()

    router.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "ok": true,
        });
    })

    router.GET("/thinfoil/info", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "public_key":    Config.publicKey,
            "endpoint":      Config.Endpoint,
        });
    })

    var wsupgrader = websocket.Upgrader{} // use default options
    router.GET("/thinfoil/peers", func(c *gin.Context) {

        if !websocket.IsWebSocketUpgrade(c.Request) {

            kvpairs, _, err := consul.KV().List("thinfoil/" + url.PathEscape(Config.Cluster) + "/peers/", (&consulapi.QueryOptions{
            }).WithContext(c.Request.Context()));

            peers := make([]map[string]interface{}, 0);
            for _,v := range kvpairs {
                val := make(map[string]interface{});
                err = json.Unmarshal(v.Value, &val);
                if err != nil { continue }
                peers = append(peers, val);
            }

            c.JSON(http.StatusOK, gin.H{
                "networks": Config.Networks,
                "peers":    peers,
            });

            return;
        }

        ws, err := wsupgrader.Upgrade(c.Writer, c.Request, nil)
        if err != nil { panic(err); }
        defer ws.Close()


        ctx2, cancel := context.WithCancel(c.Request.Context());
        defer cancel();
        go func() {
            for {
                _, _, err := ws.ReadMessage();
                if err != nil {
                    cancel();
                    return;
                }
            }
        }();

        var index uint64 = 0;
        for {
            kvpairs, meta, err := consul.KV().List("thinfoil/" + url.PathEscape(Config.Cluster) + "/peers/", (&consulapi.QueryOptions{
                WaitIndex:  index,
            }).WithContext(ctx2));
            if err != nil {
                if strings.Contains(err.Error(), "context canceled") {
                    return;
                }
                panic(err)
            }

            index = meta.LastIndex

            peers := make([]map[string]interface{}, 0);
            for _,v := range kvpairs {
                val := make(map[string]interface{});
                err = json.Unmarshal(v.Value, &val);
                if err != nil { continue }
                peers = append(peers, val);
            }

            metastr, err := json.Marshal(map[string]interface{}{
                "networks": Config.Networks,
                "peers" :   peers,
            });
            if err != nil { continue }
            err = ws.WriteMessage(websocket.BinaryMessage, metastr)
            if err != nil { panic(err) }
        }

    });

    log.Fatal(router.Run( address + ":80"))



}
