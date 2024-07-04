package middleware

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"time"
)

func CheckToken(c *fiber.Ctx) error {
	return c.Next()
}

func SysLogInit(c *fiber.Ctx) error {
	return c.Next()
}

// 统一的日志格式化输出中间件
func LoggerPrint() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()
		var logMessage string
		if err != nil {
			logMessage = fmt.Sprintf("[%s] %s %s - %s ==> [Error] %s\n", start.Format("2006-01-02 15:04:05"), c.Method(), c.Path(), time.Since(start), err.Error())
		} else {
			logMessage = "Success"
		}
		fmt.Printf(logMessage)
		return err
	}
}
