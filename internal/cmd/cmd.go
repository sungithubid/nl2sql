package cmd

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"

	"nl2sql/internal/controller/hello"
	"nl2sql/internal/controller/nl2sql"

	// 触发 logic 包的 init() 注册到 service 层
	_ "nl2sql/internal/logic/nl2sql"
	"nl2sql/internal/service"
)

var (
	Main = gcmd.Command{
		Name:  "main",
		Usage: "main",
		Brief: "start http server",
		Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {
			// 通过 service 接口初始化 NL2SQL 服务
			cleanup, err := service.Nl2sql().Init(ctx)
			if err != nil {
				g.Log().Warningf(ctx, "NL2SQL service initialization failed: %v", err)
				g.Log().Warning(ctx, "NL2SQL endpoints will return errors until the service is properly configured")
			}
			if cleanup != nil {
				defer cleanup()
			}

			s := g.Server()
			s.Group("/", func(group *ghttp.RouterGroup) {
				group.Middleware(ghttp.MiddlewareHandlerResponse)
				group.Bind(
					hello.NewV1(),
				)
			})
			// NL2SQL API 路由组
			s.Group("/api/v1", func(group *ghttp.RouterGroup) {
				group.Middleware(ghttp.MiddlewareHandlerResponse)
				group.Bind(
					nl2sql.NewV1(),
				)
			})
			s.Run()
			return nil
		},
	}
)
