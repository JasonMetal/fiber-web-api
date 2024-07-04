package sys

import (
	"encoding/json"
	"fiber-web-api/internal/app/common/config"
	"fiber-web-api/internal/app/common/utils"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"strconv"
	"strings"
	"time"
)

// ======================================= 登录相关 =======================================

// 用户登录：user 用户信息 loginType 登录类型 expire 有效期
func (user *SysUser) Login(loginType string, expire time.Duration) string {
	str := utils.MD5(user.UserName) // 用户名md5加密
	// 设置登录类型前缀
	if len(loginType) > 0 {
		str = loginType + "_" + str
	}
	// 删除所有以当前用户名开头的key
	keys, _, _ := config.RedisConn.Scan(uint64(0), config.CachePrefix+str+"*", 1000).Result()
	for i := range keys {
		config.RedisConn.Del(keys[i])
	}
	token := str + utils.GenerateRandomToken(32) // 生成token
	user.Token = token
	userJson, _ := json.Marshal(user)
	var expireTime float64 = -1 // token的有效时长
	if expire > 0 {
		expireTime = expire.Seconds()
	}
	loginMap := map[string]any{
		"createTime": time.Now().Unix(),
		"user":       string(userJson),
		"expire":     expireTime,
	}
	// 将用户信息map设置到redis中
	config.RedisConn.HMSet(config.CachePrefix+token, loginMap)
	// 设置有效期
	if expire > 0 {
		config.RedisConn.Expire(config.CachePrefix+token, expire)
	}
	// 判断当前用户部门是否存在数据权限设置
	exists, _ := config.RedisConn.Exists(config.DATA_SCOPE + user.DeptId).Result()
	if exists == 0 {
		SetDataScope(user.DeptId) // 如果没有，则需要设置
	}
	return token
}

// 获取请求头中携带的token，并解密、校验
func GetToken(c *fiber.Ctx) (string, *config.Result) {
	token := c.Get(config.TokenHeader)
	// TODO 项目中需要放开下面的代码，配合前端请求头传过来的token和sign进行解析校验
	/*sign := c.Get(config.Sign) // 这个是长度为16位的时间戳：前13位是毫秒级的时间戳，后面补3个0
	if token == "" || sign == "" {
		return "", config.ErrorCode(1003, "用户未登录")
	}
	fmt.Println("token=====>>>", token)
	fmt.Println("sign=====>>>", sign)
	// 校验前端传过来的sign和当前时间戳的差值，是否大于120秒
	currentTime := time.Now().UnixMilli() // 获取当前毫秒级时间戳
	signTime, err := strconv.ParseInt(sign[:13], 10, 64)
	if err != nil {
		mylog.Error(err.Error())
		return "", config.ErrorCode(-1, config.UNKNOWN_EXCEPTION)
	}
	diffSeconds := math.Abs(float64(currentTime-signTime)) / 1000 // 计算两个时间戳之间相差的秒数
	fmt.Println("diffSeconds=====>>>", diffSeconds)
	if diffSeconds > 120 {
		return "", config.ErrorCode(-1, "令牌超时！")
	}
	// token解密
	decrypt, err := util.AESDecrypt(token, sign, sign)
	if err != nil {
		mylog.Error("解密算法异常 AESDecrypt() 解密方法，异常信息：" + err.Error())
		return "", config.ErrorCode(-1, "解密算法异常！")
	}*/
	// TODO 这里因为没有前端，所以直接用token（正式使用时，把上面那段代码放开，下面这句删掉）
	decrypt := token
	// 校验携带的token在redis中是否存在
	exists, _ := config.RedisConn.Exists(config.CachePrefix + decrypt).Result()
	if exists == 0 {
		return "", config.ErrorCode(1003, "用户未登录")
	}
	return decrypt, nil
}

// 用户退出
func Logout(c *fiber.Ctx) *config.Result {
	token, err := GetToken(c)
	if err != nil {
		return err
	}
	config.RedisConn.Del(config.CachePrefix + token)
	return nil
}

// 获取当前用户的剩余有效时长，返回秒数，返回 -2 时，key已过期
func GetTimeOut(token string) int {
	// 使用 TTL 命令获取 key 的剩余有效时长，如果 key 不存在或已过期，TTL 将返回 -2
	ttl, err := config.RedisConn.TTL(config.CachePrefix + token).Result()
	if err != nil {
		return -2
	}
	return int(ttl.Seconds())
}

// 获取当前用户
func GetLoginUser(token string) *SysUser {
	val, _ := config.RedisConn.HGet(config.CachePrefix+token, "user").Result()
	user := SysUser{}
	json.Unmarshal([]byte(val), &user)
	// 判断当前用户部门是否存在数据权限设置
	exists, _ := config.RedisConn.Exists(config.DATA_SCOPE + user.DeptId).Result()
	if exists == 0 {
		SetDataScope(user.DeptId) // 如果没有，则需要设置
	}
	dataScope := config.RedisConn.HGetAll(config.DATA_SCOPE + user.DeptId).Val()
	user.AncestorId = dataScope["ancestorId"]
	user.AncestorName = dataScope["ancestorName"]
	user.ChildId = dataScope["childId"]
	user.ChildName = dataScope["childName"]
	return &user
}

// 获取当前用户id
func GetLoginId(token string) *string {
	user := GetLoginUser(token)
	return &user.Id
}

// 获取当前用户token的创建时间
func GetCreateTime(token string) int64 {
	val, _ := config.RedisConn.HGet(config.CachePrefix+token, "createTime").Result()
	r, _ := strconv.ParseInt(val, 10, 64)
	return r
}

// 获取当前用户token设置的有效期
func GetExpire(token string) time.Duration {
	val, _ := config.RedisConn.HGet(config.CachePrefix+token, "expire").Result()
	r, _ := strconv.ParseFloat(val, 64)
	if r == -1 {
		return -1
	}
	expire := time.Second * time.Duration(r)
	return expire
}

// 获取当前用户的所有权限集合
func GetPermList(roleId string) []string {
	redis := config.RedisConn
	val := redis.Exists(config.RolePermList).Val()
	if val == 0 {
		// redis中不存在，则添加
		permList := GetPermsAll()
		result := make(map[string]interface{})
		for key, v := range permList {
			result[key] = strings.Join(v, ";")
		}
		redis.HMSet(config.RolePermList, result)
		redis.Expire(config.RolePermList, time.Second*604800)
		return permList[roleId]
	} else {
		// redis中存在，直接从redis中拿
		data := redis.HGet(config.RolePermList, roleId).Val()
		permList := []string{}
		split := strings.Split(data, ";")
		for i := range split {
			permList = append(permList, split[i])
		}
		return permList
	}
}

// 刷新过期时间
func UpdateTimeOut(token string, expire time.Duration) {
	if expire.Seconds() < 0 {
		// -1 永不过期，Persist 将删除key的过期时间，使其永不过期
		config.RedisConn.Persist(config.CachePrefix + token)
	} else {
		config.RedisConn.Expire(config.CachePrefix+token, expire)
	}
}

// 更新用户信息
func (user *SysUser) UpdateUser(token string) {
	config.RedisConn.HSet(config.CachePrefix+token, "user", user)
}

// ======================================= 数据权限相关 =======================================

// 设置当前部门的数据范围
func SetDataScope(deptId string) {
	if deptId != "" {
		dept := SysDept{}
		dept.Id = deptId
		// 这里的数据权限条件存了部门id和名称，如果没有特殊要求的话，只用部门id也可以的。
		// 但是因为我的项目的业务原因，需要用到部门名称来过滤数据（因为有的表的数据判断是哪个部门的数据，用的不是部门id而是部门名称）
		childId, childName := GetDeptChild(deptId)     // 当前部门及子部门id和名称
		ancestorId, ancestorName := dept.GetAncestor() // 当前部门祖级id和名称
		dataScope := map[string]any{
			"ancestorId":   ancestorId,
			"ancestorName": ancestorName,
			"childId":      childId,
			"childName":    childName,
		}
		// 将数据范围信息map设置到redis中，并设置有效期为2小时
		config.RedisConn.HMSet(config.DATA_SCOPE+dept.Id, dataScope)
		config.RedisConn.Expire(config.DATA_SCOPE+dept.Id, time.Second*7200)
	}
}

// 获取数据范围条件
func GetDataScope(token string, ignoreAdmin, isId bool) string {
	if token == "" {
		return ""
	}
	loginUser := GetLoginUser(token)
	// ignoreAdmin=true 表示不管是不是管理员，都要过滤数据; ignoreAdmin=false 表示只有非管理员角色才需要过滤数据
	if ignoreAdmin || (!ignoreAdmin && loginUser.RoleKey != "CJGLY") {
		if isId {
			return loginUser.ChildId
		} else {
			return loginUser.ChildName
		}
	}
	return ""
}

// 统一的数据过滤 fieldName 要查询的字段，ignoreAdmin 是否忽略超级管理员（true 忽略 false 不忽略），isId 表示是用id还是用name查询
// dataScope 数据范围（1 所有数据 2 所在部门及子部门数据 3 所在部门数据 4 仅本人数据 5 自定义数据）
func AppendQueryDataScope(token, fieldName, dataScope string, ignoreAdmin, isId bool) string {
	str := GetDataScope(token, ignoreAdmin, isId)
	sql := ""
	if str != "" {
		// 根据 当前用户的数据范围 拼接查询条件语句 scope 数据范围、过滤条件、fieldName 查询的字段名
		if dataScope == "5" {
			// 自定义数据范围（暂不需要）
			// 5 和其他数字的范围取并集，用 or 连接，并且它们的外层不要忘了用括号括起来
		} else if dataScope == "2" {
			// 所在部门及子部门数据（用FIND_IN_SET查询）
			sql = fmt.Sprintf("FIND_IN_SET(%s,'%s')", fieldName, str)
		} else if dataScope == "3" {
			// 所在部门数据（用等于查询）
			sql = fmt.Sprintf("%s = '%s'", fieldName, str)
		} else if dataScope == "4" {
			// 仅本人数据直接用等于查询
			sql = fmt.Sprintf("%s = '%s'", fieldName, str)
		}
	}
	return sql
}

// 校验是否有数据权限（新增、修改、删除数据时）：verified 需要校验的值
func CheckDataScope(token, verified string, ignoreAdmin, isId bool) bool {
	scope := GetDataScope(token, ignoreAdmin, isId)
	// 当scope不是空值时，判断需要校验的值是否包含在scope中，不包含说明没有权限
	if scope != "" && !utils.IsContain(strings.Split(scope, ","), verified) {
		return false
	}
	return true
}
