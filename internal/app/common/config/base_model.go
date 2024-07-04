package config

import (
	"github.com/gofiber/fiber/v2"
	"time"
)

// ===================================== 公共常量 =====================================
const (
	CachePrefix       = "go-web:login:"                                                  // 缓存前缀
	ERROR_COUNT       = "go-web:errorCount:"                                             // 密码错误次数缓存key
	TokenHeader       = "go-web"                                                         // request请求头属性
	Sign              = "sign"                                                           // request请求头属性
	TokenExpire       = time.Second * 1800                                               // token默认有效期（单位秒）
	RolePermList      = "go-web:rolePermList:"                                           // 角色对应的权限列表
	UNKNOWN_EXCEPTION = "未知异常"                                                           // 全局异常 未知异常
	PARENT_VIEW       = "ParentView"                                                     // ParentView组件标识
	InitPassword      = "123456"                                                         // 初始密码
	RandomCharset     = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789" // 随机字符串
	RandomCaptcha     = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"                               // 验证码字符串
	DATA_SCOPE        = "go-web:dataScope:"                                              // 数据范围缓存
)

// ==================================== 公共model ====================================

// 一个公共的model
type BaseModel struct {
	Id         string    `query:"id" json:"id" form:"id"`
	CreatorId  *string   `json:"creatorId" form:"creatorId"`
	CreateTime time.Time `json:"createTime" form:"createTime"`
	UpdateId   *string   `json:"updateId" form:"updateId"`
	UpdateTime *string   `json:"updateTime" form:"updateTime"`
	Token      string    `gorm:"-" json:"token" form:"token"` // token
}

// 统一的返回参数格式
type Result struct {
	Code    int    `json:"code"`    // 统一的返回码，0 成功 -1 失败
	Message string `json:"message"` // 统一的返回信息
	Data    any    `json:"data"`    // 统一的返回数据
}

// 统一的树形结构格式
/*type TreeVo struct {
	Id       string   `json:"id"`       // 统一的返回码，0 成功 -1 失败
	Label    string   `json:"label"`    // 统一的返回信息
	ParentId string   `json:"parentId"` // 统一的返回数据
	Children []TreeVo `json:"children"` // 子级数据
}*/

// 将list转为统一的树形结构格式
/*func ConvertToTreeVo(list []interface{}) []TreeVo {
	result := []TreeVo{}
	for _, t := range list {
		item, _ := t.(map[string]interface{})
		id, _ := item["id"].(string)
		parentId, _ := item["parentId"].(string)
		label, _ := item["name"].(string)
		children, _ := item["children"].([]interface{})
		tree := TreeVo{Id: id, ParentId: parentId, Label: label}
		tree.Children = ConvertToTreeVo(children)
		result = append(result, tree)
	}
	return result
}*/

// 请求成功的默认返回
func Success(obj any) *Result {
	return &Result{0, "ok", obj}
}

// 请求失败的默认返回，code默认为-1
func Error(message string) *Result {
	return ErrorCode(-1, message)
}

// 请求失败的默认返回
func ErrorCode(code int, message string) *Result {
	return &Result{code, message, nil}
}

// 分页结构体封装
type PageInfo struct {
	List  any   `json:"list"`  // 返回结果
	Total int64 `json:"total"` // 返回总数
}

// 定义一个结构体，用于扩展接口路由信息
type CustomApi struct {
	Group       string        // 所属组
	Method      string        // 请求方法
	Path        string        // 接口地址
	Description string        // 接口描述
	Permission  string        // 权限标识（没有限制时留空，多个用 ; 号分隔，表示满足任意一个即可）
	HandlerFunc fiber.Handler // 请求处理函数
}

// 将路由信息存储到map中，path为key
var RouteApi = map[string]CustomApi{}
