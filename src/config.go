package src

import (
    "os"
    "encoding/json"
    "log"
    "golang.zx2c4.com/wireguard/wgctrl/wgtypes"
    "net"
)



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
    PublicKey       string `json:"public_key,omitempty"`
    PresharedKey    string `json:"preshared_key,omitempty"`
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

        psk, err := wgtypes.GenerateKey();
        if err != nil {
            log.Fatal(err);
        }

        Config.Cluster      = "starfeld";
        Config.Networks     = []string{"169.254.0.0/16"}
        Config.PrivateKey   = secret.String();
        Config.PublicKey    = secret.PublicKey().String();
        Config.Endpoint     = getOutboundIP() + ":52525"
        Config.PresharedKey = psk.String();
        Config.Routes       = []string{"169.254.1.2/32"};

        log.Printf("created new config.json, make preshared_key the same everywhere and change address to a free ip\n\n");

        err = json.NewEncoder(file).Encode(&Config);
        if err != nil {
            log.Fatal(err);
        }

        file.Seek(0,0);
    }
    defer file.Close()

    err = json.NewDecoder(file).Decode(&Config);
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

