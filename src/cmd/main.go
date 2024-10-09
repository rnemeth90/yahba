package main

import (
    "fmt"
    "github.com/spf13/pflag"
)

func main() {
    var name string
    pflag.StringVarP(&name, "name", "n", "", "Your name")
    pflag.Parse()

    if name == "" {
        fmt.Println("Please provide your name using the --name flag.")
        return
    }

    fmt.Printf("Hello, %s!\n", name)
}
