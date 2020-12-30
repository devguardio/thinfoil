package src

import (
    "os"
    "encoding/json"
    "log"
    "golang.zx2c4.com/wireguard/wgctrl/wgtypes"
    "net"
    "fmt"
)



type BootstrapPeer struct {
    Endpoint        string      `json:"endpoint,omitempty"`
    Routes          []string    `json:"routes,omitempty"`
    PublicKey       string      `json:"public_key,omitempty"`
}

type ConfigT struct {
    // name of interface and k/v tree
    Cluster         string `json:"cluster,omitempty"`
    // list of networks to route through the interface by default.
    // to prevent leakage due to transient route failure,
    // routes are not added individually
    Networks        []string `json:"networks,omitempty"`
    Endpoint        string `json:"endpoint,omitempty"`
    Routes          []string `json:"routes,omitempty"`
    PrivateKey      string `json:"private_key,omitempty"`
    PresharedKey    string `json:"preshared_key,omitempty"`
    BootstrapPeers  []BootstrapPeer `json:"bootstrap_peers"`

    publicKey       string

}

var Config ConfigT;

func ConfigLoad() {
    file, err := os.Open("config.json")
    if err != nil {
        file, err = os.OpenFile("config.json", os.O_RDWR|os.O_CREATE, 0600)
        if err != nil {
            log.Fatal("cannot create config.json : ", err);
            return;
        }

        var secret, err = wgtypes.GeneratePrivateKey();
        if err != nil {
            log.Fatal(err);
        }

        Config.publicKey  = secret.PublicKey().String();

        psk, err := wgtypes.GenerateKey();
        if err != nil {
            log.Fatal(err);
        }

        Config.Cluster      = "starfeld";
        Config.Networks     = []string{"172.27.0.0/15"}
        Config.PrivateKey   = secret.String();
        Config.Endpoint     = getOutboundIP() + ":52525"
        Config.PresharedKey = psk.String();
        Config.Routes       = []string{"172.27.0.10/32"};
        Config.BootstrapPeers = []BootstrapPeer{}


        jw := json.NewEncoder(file);
        jw.SetIndent("", " ");
        err = jw.Encode(&Config);
        if err != nil {
            log.Fatal(err);
        }

        file.Seek(0,0);

        bs, err := json.Marshal(&BootstrapPeer{
            Endpoint:   Config.Endpoint,
            Routes:     Config.Routes,
            PublicKey:  Config.publicKey,
        });
        if err != nil {
            log.Fatal(err);
        }

        fmt.Println("created new config.json, make preshared_key the same everywhere and change routes to a unique ip");
        fmt.Println("put this into the other peers config.json under bootstrap_peers, with route changed as well:");
        fmt.Println("");
        fmt.Println(string(bs));
        os.Exit(0);
    }
    defer file.Close()

    err = json.NewDecoder(file).Decode(&Config);
    if err != nil {
        log.Fatal(err);
    }

    secret , err := wgtypes.ParseKey(Config.PrivateKey);
    if err != nil { panic(fmt.Errorf("config.json private_key: %w", err)) }
    Config.publicKey  = secret.PublicKey().String();
}

func ConfigStore() {
    file, err := os.OpenFile("config.json", os.O_RDWR|os.O_CREATE, 0600)
    if err != nil {
        log.Fatal("cannot create config.json : ", err);
        return;
    }
    defer file.Close();

    jw := json.NewEncoder(file);
    jw.SetIndent("", " ");
    err = jw.Encode(&Config);
    if err != nil {
        log.Fatal(err);
    }
}


func getOutboundIP() string {
    conn, err := net.Dial("udp", "8.8.8.8:80")
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    localAddr := conn.LocalAddr().(*net.UDPAddr)

    return localAddr.IP.String()
}

