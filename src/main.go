package src;

import (
    "github.com/spf13/cobra"
    "log"
    "os"
);

var rootCmd cobra.Command;


func init() {
    log.SetFlags(log.Lshortfile);

    rootCmd = cobra.Command {
        Use:        "thinfoil",
        Short:      "starfeld networks\n",
        Version:    "1",
    }
    rootCmd.SetVersionTemplate("{{printf \"%s\\n\" .Version}}");

    rootCmd.AddCommand(&cobra.Command{
        Use:    "server",
        Short:  "start a thinfoil server with consul backend",
        Run: func(cmd *cobra.Command, args []string) {
            Server();
        },
    });
    rootCmd.AddCommand(&cobra.Command{
        Use:    "client",
        Short:  "start a thinfoil client",
        Run: func(cmd *cobra.Command, args []string) {
            Client();
        },
    });
}

func Main() {

    if err := rootCmd.Execute(); err != nil {
        os.Exit(1);
    }
}
