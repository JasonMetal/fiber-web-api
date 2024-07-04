package sys

import (
	"fiber-web-api/internal/app/common/config"
	"fiber-web-api/internal/app/common/utils"
	"fiber-web-api/internal/app/model/sys"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/pkg/errors"
	"time"
)

type LoginController struct{}

// 获取公钥
func (LoginController) GetKey(c *fiber.Ctx) error {
	ip := c.IP()
	currentTime := time.Now().Unix()
	err, _ := lockedUser(currentTime, ip, "IP") //判断ip是否锁定
	if err != nil {
		return c.Status(200).JSON(config.ErrorCode(1004, err.Error()))
	}
	return c.Status(200).JSON(config.Success(utils.GetPublicKey()))
}

// 获取验证码
func (LoginController) GetCode(c *fiber.Ctx) error {
	id, base64 := utils.GenerateCaptcha(4, 100, 42)
	code := make(map[string]string)
	code["codeId"] = id
	code["code"] = base64
	return c.Status(200).JSON(config.Success(code))
}

// 登录
func (LoginController) Login(c *fiber.Ctx) error {
	ip := c.IP()
	//currentTime := time.Now().Unix()
	//err, _ := lockedUser(currentTime, ip, "IP") //判断ip是否锁定
	//if err != nil {
	//	return c.Status(200).JSON(config.ErrorCode(1004, err.Error()))
	//}
	//code := c.FormValue("code")
	//codeId := c.FormValue("codeId")
	userName := c.FormValue("userName")
	password := c.FormValue("password")
	// 解密
	//userName = util.RSADecrypt(userName)
	//password = util.RSADecrypt(password)
	log.Debug(fmt.Sprintf("用户名：%s", userName))
	log.Debug(fmt.Sprintf("password：%s", password))
	// 校验验证码是否正确
	//b := util.CaptVerify(codeId, code)
	//if !b {
	//	return c.Status(200).JSON(config.Error("验证码错误或已过期"))
	//}
	var syslog = sys.SysLog{IP: ip, Title: "用户登录", Type: "登录", Method: "login", Url: "/sys/login", State: "登录成功"}
	syslog.CreatorId = &userName
	// 校验用户名和密码
	safe := sys.SysSafe{}
	safe.GetById()
	user, result := passwordErrorNum(ip, userName, password, safe)
	if result.Code != 0 {
		syslog.State = "登录失败"
		syslog.Info = result.Message
		syslog.Insert()
		return c.Status(200).JSON(result)
	}
	i := safe.IdleTimeSetting //如果系统闲置时间为0，设置token和session永不过期
	// 登录
	token := ""
	if i == 0 {
		token = user.Login("", -1) // 永不过期
	} else {
		token = user.Login("", config.TokenExpire) // 默认保持登录为30分钟
	}
	syslog.Info = userName + "登录成功"
	syslog.Insert()
	return c.Status(200).JSON(config.Success(token))
}

// 退出
func (LoginController) Logout(c *fiber.Ctx) error {
	err := sys.Logout(c) // 退出登录
	if err != nil {
		return c.Status(200).JSON(err)
	}
	return c.Status(200).JSON(config.Success(nil))
}

// 判断账号或IP是否锁定
func lockedUser(currentTime int64, userName, msg string) (error, bool) {
	flag := false
	// 如果没有错误次数，直接返回
	if exists, _ := config.RedisConn.Exists(config.ERROR_COUNT + userName).Result(); exists == 0 {
		return nil, flag
	}
	loginTime, _ := config.RedisConn.HGet(config.ERROR_COUNT+userName, "loginTime").Int64()
	isLocaked, _ := config.RedisConn.HGet(config.ERROR_COUNT+userName, "isLocaked").Result()
	if "true" == isLocaked && currentTime < loginTime {
		diff := loginTime - currentTime // 计算时间差
		minutes := int(diff / 60)       // 将差值转换为分钟
		err := fmt.Sprintf("%s锁定中，还没到允许登录的时间，请%d分钟后再尝试", msg, minutes)
		return errors.New(err), flag
	} else {
		flag = true
		config.RedisConn.HSet(config.ERROR_COUNT+userName, "isLocaked", "false") //重置为false
	}
	return nil, flag
}

// 校验账号、密码、ip
func passwordErrorNum(ip, userName, password string, safe sys.SysSafe) (*sys.SysUser, *config.Result) {
	currentTime := time.Now().Unix() // 获取当前时间的时间戳（单位：秒）
	//错误3次，锁定15分钟后才可登陆 允许时间加上定义的登陆时间（毫秒）
	timeStamp := currentTime + 900
	errorCount := 3
	lockDuration := 15
	//密码登录限制（0：连续错3次，锁定账号15分钟。1：连续错5次，锁定账号30分钟）
	if safe.PwdLoginLimit == 1 {
		timeStamp = currentTime + 1800
		errorCount = 5
		lockDuration = 30
	}
	user := sys.SysUser{}
	user.UserName = userName

	//查询用户
	err := user.GetUser()
	if err != nil || user.Id == "" {
		return nil, checkIPLocked(ip, currentTime, timeStamp, errorCount, lockDuration)
	}
	//判断账号是否锁定
	err, flag := lockedUser(currentTime, userName, "账号")
	if err != nil {
		return nil, config.ErrorCode(1004, err.Error())
	}
	//根据前端输入的密码（明文），和加密的密码、盐值进行比较，判断输入的密码是否正确
	authenticate := utils.AuthenticatePassword(password, user.Password)
	if authenticate {
		//密码正确错误次数清零
		config.RedisConn.Del(config.ERROR_COUNT + userName)
		config.RedisConn.Del(config.ERROR_COUNT + ip)
	} else {
		return nil, checkNameLocked(ip, userName, timeStamp, errorCount, lockDuration, flag)
	}
	return &user, config.Success(nil)
}

// 校验ip是否锁定
func checkIPLocked(ip string, currentTime, timeStamp int64, errorCount, lockDuration int) *config.Result {
	redis := config.RedisConn
	//判断ip是否锁定
	err, flag := lockedUser(currentTime, ip, "IP")
	if err != nil {
		return config.ErrorCode(1004, err.Error())
	}
	exists, _ := redis.Exists(config.ERROR_COUNT + ip).Result()
	if exists == 0 { // 键不存在，第一次登录
		loginMap := map[string]any{
			"errorNum":  1,
			"loginTime": timeStamp,
			"isLocaked": "false", // 是否锁定，默认为false
		}
		redis.HMSet(config.ERROR_COUNT+ip, loginMap)
	} else {
		i, _ := redis.HGet(config.ERROR_COUNT+ip, "errorNum").Int()
		if flag && i == errorCount { // 当错误次数达到限定次数时，走到这一步说明已经过了锁定时间再次登录，这时重新将错误次数设置为1
			redis.HSet(config.ERROR_COUNT+ip, "errorNum", 1)
		} else {
			redis.HIncrBy(config.ERROR_COUNT+ip, "errorNum", 1)
		}
		redis.HSet(config.ERROR_COUNT+ip, "loginTime", timeStamp)
	}
	i, _ := redis.HGet(config.ERROR_COUNT+ip, "errorNum").Int()
	if i == errorCount {
		// 将锁定状态改为true表示已锁定
		redis.HSet(config.ERROR_COUNT+ip, "isLocaked", "true")
		return config.ErrorCode(1004, fmt.Sprintf("用户名或密码错误%d次，现已被锁定，请%d分钟后再尝试", errorCount, lockDuration))
	}
	return config.ErrorCode(1000, fmt.Sprintf("用户名或密码错误，总登录次数%d次，剩余次数: %d", errorCount, (errorCount-i)))
}

// 校验用户名是否锁定
func checkNameLocked(ip, userName string, timeStamp int64, errorCount, lockDuration int, flag bool) *config.Result {
	redis := config.RedisConn
	exists, _ := redis.Exists(config.ERROR_COUNT + userName).Result()
	if exists == 0 { // 键不存在，第一次登录
		loginMap := map[string]any{
			"errorNum":  1,
			"loginTime": timeStamp,
			"isLocaked": "false", // 是否锁定，默认为false
		}
		redis.HMSet(config.ERROR_COUNT+userName, loginMap)
		if exists, _ = redis.Exists(config.ERROR_COUNT + ip).Result(); exists == 0 {
			redis.HMSet(config.ERROR_COUNT+ip, loginMap)
		}
	} else {
		i1, i2 := 0, 0
		if exists, _ = redis.Exists(config.ERROR_COUNT + userName).Result(); exists > 0 {
			i1, _ = redis.HGet(config.ERROR_COUNT+userName, "errorNum").Int()
		}
		if exists, _ = redis.Exists(config.ERROR_COUNT + ip).Result(); exists > 0 {
			i2, _ = redis.HGet(config.ERROR_COUNT+ip, "errorNum").Int()
		} else {
			redis.HSet(config.ERROR_COUNT+ip, "isLocaked", "false")
		}
		// 每一次错误，同时记录当前IP和用户名的错误次数
		if flag && (i1 == errorCount || i2 == errorCount) { // 走到这一步说明已经过了锁定时间再次登录，这时重新将错误次数设置为1
			if i1 > i2 { // i1 > i2 是用户名错误次数到达限定次数，将用户名的错误次数重置为1
				redis.HSet(config.ERROR_COUNT+userName, "errorNum", 1)
			} else if i2 > i1 { // i2 > i1 是IP错误次数到达限定次数，将IP的错误次数重置为1
				redis.HSet(config.ERROR_COUNT+ip, "errorNum", 1)
			} else { // 否则就是用户名和IP错误次数相等，将两个的错误次数同时重置为1
				redis.HSet(config.ERROR_COUNT+userName, "errorNum", 1)
				redis.HSet(config.ERROR_COUNT+ip, "errorNum", 1)
			}
		} else {
			redis.HIncrBy(config.ERROR_COUNT+userName, "errorNum", 1)
			redis.HIncrBy(config.ERROR_COUNT+ip, "errorNum", 1)
		}
		redis.HSet(config.ERROR_COUNT+userName, "loginTime", timeStamp)
		redis.HSet(config.ERROR_COUNT+ip, "loginTime", timeStamp)
	}
	i1, _ := redis.HGet(config.ERROR_COUNT+userName, "errorNum").Int()
	i2, _ := redis.HGet(config.ERROR_COUNT+ip, "errorNum").Int()
	i := i1                                   // i默认为i1，需要取i1、i2两个中，较大的那一个
	if i1 == errorCount || i2 == errorCount { // 任意一个满足，将值大的那个设置为锁定
		if i1 > i2 { // i1 > i2 是用户名错误次数到达限定次数，将用户名的锁定状态设置为锁定
			redis.HSet(config.ERROR_COUNT+userName, "isLocaked", "true")
		} else if i2 > i1 { // i2 > i1 是IP错误次数到达限定次数，将IP的锁定状态设置为锁定
			i = i2
			redis.HSet(config.ERROR_COUNT+ip, "isLocaked", "true")
		} else { // 否则就是用户名和IP错误次数相等，将两个的锁定状态同时设置为锁定
			redis.HSet(config.ERROR_COUNT+userName, "isLocaked", "true")
			redis.HSet(config.ERROR_COUNT+ip, "isLocaked", "true")
		}
		return config.ErrorCode(1004, fmt.Sprintf("用户名或密码错误%d次，现已被锁定，请%d分钟后再尝试", errorCount, lockDuration))
	}
	return config.ErrorCode(1000, fmt.Sprintf("用户名或密码错误，总登录次数%d次，剩余次数: %d", errorCount, (errorCount-i)))
}
