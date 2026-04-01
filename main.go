package main

import (
	_ "nl2sql/internal/packed"

	"github.com/gogf/gf/v2/os/gctx"

	"nl2sql/internal/cmd"

	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
)

func main() {
	cmd.Main.Run(gctx.GetInitCtx())
}
