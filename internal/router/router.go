package router

import (
	"fiber-web-api/internal/app/common/config"
	api "fiber-web-api/internal/app/controller/sys"
	"github.com/gofiber/fiber/v2"
)

func InitRouter() *fiber.App {
	// 配置路由
	app := fiber.New()
	// init yaml conf
	config.InitConfig()
	apis := InitApi()
	for _, api := range apis {
		if api.Method == "POST" {
			app.Post(api.Path, api.HandlerFunc)
		}
		if api.Method == "GET" {
			app.Get(api.Path, api.HandlerFunc)
		}
		if api.Method == "DELETE" {
			app.Delete(api.Path, api.HandlerFunc)
		}
		config.RouteApi[api.Path] = api
	}

	////test
	//app.Get("/", func(c *fiber.Ctx) error {
	//	return c.SendString("Hello, World!")
	//})
	return app
}

var (
	login = api.LoginController{}
	log   = api.LogController{}
	safe  = api.SafeController{}
	user  = api.UserController{}
	dept  = api.DeptController{}
	role  = api.RoleController{}
	menu  = api.MenuController{}
	dict  = api.DictController{}
)

// 初始化接口路由api
func InitApi() []config.CustomApi {
	return []config.CustomApi{
		// 登录路由
		{"登录", "GET", "/sys/getKey", "获取RSA公钥", "", login.GetKey},
		{"登录", "GET", "/sys/getCode", "获取验证码", "", login.GetCode},
		{"登录", "POST", "/sys/login", "用户登录", "", login.Login},
		{"登录", "DELETE", "/sys/logout", "用户退出", "", login.Logout},
		// 日志管理
		{"日志管理", "GET", "/sys/log/list", "日志列表", "system:userLog:view", log.GetPage},
		// 安全设置
		{"安全设置", "GET", "/sys/safe/getSafeSet", "获取安全设置", "system:userLog:view", safe.GetSafeSet},
		{"安全设置", "POST", "/sys/safe/update", "修改安全设置", "system:safe:update", safe.Update},
		// 用户管理
		{"用户管理", "GET", "/sys/user/getLoginUser", "获取当前登录的用户", "", user.GetLoginUser},
		{"用户管理", "GET", "/sys/user/list", "用户列表", "system:user:view", user.GetPage},
		{"用户管理", "GET", "/sys/user/getById/:id", "根据id获取用户", "system:user:view", user.GetById},
		{"用户管理", "POST", "/sys/user/insert", "新增用户", "system:user:add", user.Insert},
		{"用户管理", "POST", "/sys/user/update", "修改用户", "system:user:update", user.Update},
		{"用户管理", "DELETE", "/sys/user/delete", "删除用户", "system:user:delete", user.Delete},
		{"用户管理", "POST", "/sys/user/updatePassword", "设置密码", "system:user:updatePassword", user.UpdatePassword},
		{"用户管理", "POST", "/sys/user/resetPassword", "重置密码", "system:user:updatePassword", user.ResetPassword},
		{"用户管理", "POST", "/sys/user/upload", "上传头像", "", user.Upload},
		// 部门管理
		{"部门管理", "GET", "/sys/dept/list", "部门树列表", "system:user:view;system:dept:view", dept.GetList},
		{"部门管理", "GET", "/sys/dept/getById/:id", "根据id获取部门", "system:user:view;system:dept:view", dept.GetById},
		{"部门管理", "POST", "/sys/dept/insert", "新增部门", "system:user:add;system:dept:add", dept.Insert},
		{"部门管理", "POST", "/sys/dept/update", "修改部门", "system:user:update;system:dept:update", dept.Update},
		{"部门管理", "DELETE", "/sys/dept/delete/:id", "删除部门", "system:user:delete;system:dept:delete", dept.Delete},
		{"部门管理", "GET", "/sys/dept/deptSelect", "部门下拉树列表", "", dept.GetSelectList},
		// 角色管理
		{"角色管理", "GET", "/sys/role/list", "角色列表", "system:role:view", role.GetPage},
		{"角色管理", "GET", "/sys/role/getById/:id", "根据id获取角色", "system:role:view", role.GetById},
		{"角色管理", "GET", "/sys/role/createRoleCode", "生成角色编码", "", role.CreateCode},
		{"角色管理", "POST", "/sys/role/insert", "新增角色", "system:role:add", role.Insert},
		{"角色管理", "POST", "/sys/role/update", "修改角色", "system:role:update", role.Update},
		{"角色管理", "POST", "/sys/role/updateState", "修改角色状态", "system:role:update", role.UpdateState},
		{"角色管理", "DELETE", "/sys/role/delete", "删除角色", "system:role:delete", role.Delete},
		{"角色管理", "GET", "/sys/role/roleSelect", "角色下拉框", "", role.GetSelectList},
		// 菜单管理
		{"菜单管理", "GET", "/sys/menu/list", "菜单列表", "system:menu:view", menu.GetList},
		{"菜单管理", "GET", "/sys/menu/getRouters", "路由列表", "", menu.GetRouters},
		{"菜单管理", "GET", "/sys/menu/getById/:id", "根据id获取菜单", "system:menu:view", menu.GetById},
		{"菜单管理", "GET", "/sys/menu/roleMenuTree/:roleId", "获取对应角色菜单列表树", "", menu.RoleMenuTree},
		{"菜单管理", "POST", "/sys/menu/insert", "新增菜单", "system:menu:add", menu.Insert},
		{"菜单管理", "POST", "/sys/menu/update", "修改菜单", "system:menu:update", menu.Update},
		{"菜单管理", "DELETE", "/sys/menu/delete/:id", "删除菜单", "system:menu:delete", menu.Delete},
		// 字典管理
		{"字典管理", "GET", "/sys/dict/typeList", "获取字段类型列表", "system:dict:view", dict.GetTypeList},
		{"字典管理", "GET", "/sys/dict/list", "字段项列表分页", "system:dict:view", dict.GetPage},
		{"字典管理", "GET", "/sys/dict/getById/:id", "根据id获取字段", "system:dict:view", dict.GetById},
		{"字典管理", "GET", "/sys/dict/createDictCode", "生成字典代码", "", dict.CreateCode},
		{"字典管理", "GET", "/sys/dict/hasDictByName", "字典名称是否存在", "", dict.HasByName},
		{"字典管理", "GET", "/sys/dict/hasDictByCode", "字典代码是否存在", "", dict.HasByCode},
		{"字典管理", "POST", "/sys/dict/insert", "新增字典", "system:dict:add", dict.Insert},
		{"字典管理", "POST", "/sys/dict/update", "修改字典", "system:dict:update", dict.Update},
		{"字典管理", "DELETE", "/sys/dict/deleteType/:id", "删除字典类型", "system:dict:delete", dict.DeleteType},
		{"字典管理", "DELETE", "/sys/dict/delete", "删除字典", "system:dict:delete", dict.Delete},
		{"字典管理", "GET", "/sys/dict/getByTypeCode", "根据字典类型代码获取字典项列表", "", dict.GetByTypeCode},
	}
}
