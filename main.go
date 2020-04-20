package main

import (
	"github.com/subosito/gotenv"

	"github.com/Bnei-Baruch/gxydb-api/cmd"
)

func init() {
	gotenv.Load()
}

func main() {
	cmd.Execute()
}
