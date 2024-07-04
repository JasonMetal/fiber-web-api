package main

import (
	"fiber-web-api/internal/app/common/config"
	"fmt"
)
import "fiber-web-api/internal/router"

func main() {
	app := router.InitRouter()
	//RSA密钥对
	app.Listen(fmt.Sprintf(":%d", config.HTTPPort))
}
