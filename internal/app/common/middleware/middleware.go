package middleware

import (
	"fiber-web-api/internal/app/common/config"
	"fiber-web-api/internal/app/common/utils"
	model "fiber-web-api/internal/app/model/sys"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// isIPInWhitelist
//
//	@Description:
//	@param ip
//	@return bool
func isIPInWhitelist(ip string) bool {
	pIp := net.ParseIP(ip)
	for _, allowedIp := range config.AuthHost {
		if allowedIp == "*" {
			return true
		}
		if strings.Contains(allowedIp, "*") {
			// 将 * 转换为正则表达式
			regexPattern := strings.ReplaceAll(allowedIp, "*", ".*")
			if match, _ := regexp.MatchString("^"+regexPattern+"$", ip); match {
				return true
			}
		} else {
			// 非通配符的精确匹配
			if pIp.Equal(net.ParseIP(allowedIp)) {
				return true
			}
		}
	}
	return false
}

// CheckToken
//
//	@Description: 验证token
//	@param c
//	@return error
func CheckToken(c *fiber.Ctx) error {
	// 获取用户请求的ip
	ip := c.IP()
	// 校验用户 IP 是否在白名单内
	if !isIPInWhitelist(ip) {
		return c.Status(http.StatusOK).JSON(config.Error("非法访问"))
	}
	// 排除指定接口，不校验token
	path := c.Path()
	if path == "/sys/login" || path == "/sys/getKey" || path == "/sys/getCode" {
		return c.Next()
	}
	// 获取请求头中的token，并校验
	token, err := model.GetToken(c)
	if err != nil {
		return c.Status(http.StatusOK).JSON(err)
	}
	// 鉴权
	if !checkPermission(c, token) {
		return c.Status(http.StatusOK).JSON(config.Error("没有操作权限"))
	}
	// 设置请求头
	setHeader(c)
	// 刷新token有效期刷新和定期刷新
	refreshToken(c, token)
	// 排除三个接口，都要经过中间件，然后这个中间件获取token时，已经解析、检验过token了
	// 所以这里直接将解析且校验通过的token重新设置到请求头中，当那些接口去拿请求头的token时，直接拿，不用再进行解析校验。
	c.Request().Header.Set(config.TokenHeader, token)
	return c.Next()
}

func refreshToken(c *fiber.Ctx, token string) {
	// 刷新有效期
	v := model.GetCreateTime(token)
	// 获取token创建时间
	// 判断token的创建时间是否大于2小时，如果是则需要刷新token
	s := time.Now().Unix() - v
	hour := s / 1000 / 3600
	if hour >= 2 {
		// TODO
		user := model.GetLoginUser(token)
		expire := model.GetExpire(token)
		// TODO
		splits := strings.Split(token, "_")
		var newToken string
		if len(splits) > 1 {
			newToken = user.Login(splits[0], expire)
		} else {
			newToken = user.Login("", expire)
		}
		token = newToken
		// 设置新的toke到 请求头中
		c.Response().Header.Set(config.TokenHeader, newToken)
	}
	// 获取token的过期时间
	timeOut := model.GetTimeOut(token)
	if timeOut != -1 {
		model.UpdateTimeOut(token, config.TokenExpire)
	}
}

func setHeader(c *fiber.Ctx) {
	// 校验Origin值
	origin := c.Get("Origin")
	if origin != "" && utils.IsContain(config.AllowedOrigins, origin) {
		c.Set("Access-Control-Allow-Origin", origin)
	} else {
		c.Set("Access-Control-Allow-Origin", "")
	}
	c.Set("Set-Cookie", "name=value; SameSite=Strict;cookiename=httponlyTest;Path=/;Domain=domainvalue;Max-Age=seconds;Secure;HTTPOnly")
	c.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; frame-ancestors 'self'; object-src 'none'")
	c.Set("Access-Control-Allow-Credentials", "true")
	c.Set("Referrer-Policy", "no-referrer")
	c.Set("X-XXS-Protection", "1; mode=block") //1; mode=block: 启用XSS保护，并检查到XSS攻击时，停止渲染页面
	c.Set("X-Content-Type-Options", "nosniff") //互联网上的资源有各种类型，通常浏览器会根据响应头的Content-Type字段来分辨它们的类型。通过这响应头可以禁用浏览器的类型猜测行为
	c.Set("X-Frame-Options", "SAMEORIGIN")     //SAMEORIGIN: 不允许被本域以外的页面嵌入
	c.Set("X-DNS-Prefetch-Control", "off")
	c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Set("Pragma", "no-cache")
	c.Set("Expires", "0")
}

var apis = config.RouteApi

func checkPermission(c *fiber.Ctx, token string) bool {
	path := c.Path()
	flag := false
	api := apis[path]
	if api.Permission != "" {
		user := model.GetLoginUser(token)
		permList := model.GetPermList(user.RoleId)
		split := strings.Split(api.Permission, ";")
		for i := range split {
			if utils.IsContain(permList, split[i]) {
				flag = true
				break
			}
		}
	}
	return flag
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
