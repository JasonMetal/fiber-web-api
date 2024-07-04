package sys

import (
	"fiber-web-api/internal/app/common/config"
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"strings"
	"time"
)

// 菜单管理
type SysMenu struct {
	config.BaseModel
	ParentId   string    `json:"parentId" form:"parentId"`     // 上级部门id
	Name       string    `json:"name" form:"name"`             // 菜单名称
	Sort       int       `json:"sort" form:"sort"`             // 排序
	Url        string    `json:"url" form:"url"`               // 访问路径
	Path       string    `son:"path" form:"path"`              // 组件名称
	Type       string    `json:"type" form:"type"`             // 菜单类型（M目录 C菜单 F按钮）
	State      int       `json:"state" form:"state"`           // 菜单状态（1正常 2停用 3删除）
	Perms      string    `json:"perms" form:"perms"`           // 权限标识
	Visible    bool      `json:"visible" form:"visible"`       // 显示状态（0隐藏  1显示）
	Icon       string    `json:"icon" form:"icon"`             // 菜单图标
	ActiveMenu string    `json:"activeMenu" form:"activeMenu"` // 菜单高亮
	IsFrame    bool      `json:"isFrame" form:"isFrame"`       // 是否外链（0 否 1 是）
	Remark     string    `json:"remark" form:"remark"`         // 备注
	Children   []SysMenu `gorm:"-" json:"children"`            // 子级数据
}

type Meta struct {
	Title      string `json:"title"`      // 设置该路由在侧边栏和面包屑中展示的名字
	Icon       string `json:"icon"`       // 菜单图标
	ActiveMenu string `json:"activeMenu"` // 菜单高亮
	RedDot     bool   `json:"redDot" `
}

// 前端路由
type Router struct {
	Name       string   `json:"name"`       // 菜单名称
	Path       string   `json:"path"`       // 组件名称
	Hidden     bool     `json:"hidden"`     // 是否隐藏路由，当设置 true 的时候该路由不会再侧边栏出现
	Redirect   string   `json:"redirect"`   // 重定向地址，当设置 noRedirect 的时候该路由在面包屑导航中不可被点击
	Component  string   `json:"component"`  // 组件地址
	AlwaysShow bool     `json:"alwaysShow"` // 当你一个路由下面的 children 声明的路由大于1个时，自动会变成嵌套的模式--如组件页面
	ActiveMenu string   `json:"activeMenu"` // 菜单高亮
	Meta       Meta     `json:"meta"`       // 其他元素
	Children   []Router `json:"children"`   // 子级数据
}

// 获取表名
func (SysMenu) TableName() string {
	return "sys_menu"
}

// 查询菜单列表的sql
var sql = `
		select a.id,parent_id,name,type,url,path,state,ifnull(perms,'') as perms,icon, sort,visible,active_menu,is_frame
        from sys_menu a
        left join sys_role_menu b on a.id = b.menu_id
`

// 树形菜单列表
func (e *SysMenu) GetList() interface{} {
	var list []SysMenu // 查询结果
	query := config.DB.Table(e.TableName())
	if e.Id != "" { // 角色id不为空，根据角色获取菜单
		where := sql + " where b.role_id = ?"
		args := []interface{}{e.Id}
		if e.Name != "" {
			where += " and name like ?"
			args = append(args, fmt.Sprintf("%%%s%%", e.Name))
		}
		if e.State != 0 {
			where += " and state = ?"
			args = append(args, e.State)
		}
		query.Debug().Order("parent_id,sort asc").Raw(where, args...).Find(&list)
	} else {
		if e.Name != "" {
			query.Where("name like ?", fmt.Sprintf("%%%s%%", e.Name))
		}
		if e.State != 0 {
			query.Where("state = ?", e.State)
		}
		query.Debug().Order("parent_id,sort asc").Find(&list)
	}
	return e.BuildTree(list, "ROOT")
}

// 获取路由（根据当前用户的角色id）
func (e *SysMenu) GetRouters() interface{} {
	var list []SysMenu // 查询结果
	where := ` where b.role_id = ? and type in ('M', 'C') and a.state = 1`
	where = sql + where
	config.DB.Table(e.TableName()).Debug().Order("parent_id,sort asc").Raw(where, e.Id).Find(&list)
	return buildMenus(e.BuildTree(list, "ROOT"))
}

// 详情
func (e *SysMenu) GetById() {
	config.DB.Table(e.TableName()).Where("id = ?", e.Id).Find(e)
}

// 根据角色id获取菜单权限标识
func GetPermsMenuByRoleId(roleId string) []string {
	var list []SysMenu
	sql := `
		select a.id,perms
        from sys_menu a
        left join sys_role_menu b on a.id = b.menu_id
        where b.role_id = ?
        order by parent_id,sort
	`
	config.DB.Raw(sql, roleId).Find(&list)
	var result []string
	for _, menu := range list {
		result = append(result, menu.Perms)
	}
	return result
}

// 新增
func (e *SysMenu) Insert() (err error) {
	var count int64
	// 校验角色名称和角色代码
	query := config.DB.Table(e.TableName())
	if e.ParentId == "0" {
		e.ParentId = "ROOT"
	}
	query.Where("name = ? and parent_id = ?", e.Name, e.ParentId).Count(&count)
	if count > 0 {
		err = errors.New("菜单名称已存在！")
		return
	}
	e.Id = strings.ReplaceAll(uuid.NewString(), "-", "")
	e.CreatorId = GetLoginId(e.Token)
	e.CreateTime = time.Now()
	config.DB.Table(e.TableName()).Create(e)
	return
}

// 修改
func (e *SysMenu) Update() (err error) {
	var count int64
	// 校验角色名称和角色代码
	query := config.DB.Table(e.TableName())
	if e.ParentId == "0" {
		e.ParentId = "ROOT"
	}
	if e.Id == e.ParentId {
		err = errors.New("上级菜单不能是自己！")
		return
	}
	query.Where("name = ? and parent_id = ? and id <> ?", e.Name, e.ParentId, e.Id).Count(&count)
	if count > 0 {
		err = errors.New("菜单名称已存在！")
		return
	}
	var m = SysMenu{}
	config.DB.Model(&SysMenu{}).Where("id = ?", e.Id).Find(&m)
	//config.DB.Model(&SysMenu{}).Select("parent_id", "name", "sort", "url", "path", "type", "state", "perms", "visible", "icon", "active_menu", "is_frame", "remark").Where("id = ?", e.Id).Save(e)
	config.DB.Model(&SysMenu{}).Omit("id", "create_time").Where("id = ?", e.Id).Save(e)
	if m.Perms != e.Perms { // 更改了权限标识，在redis缓存中也需要更改
		UpdatePerm(m.Perms, e.Perms)
	}
	return
}

// 删除
func (e *SysMenu) Delete() (err error) {
	// 1、校验是否存在下级
	var count int64
	query := config.DB.Table(e.TableName())
	query.Where("parent_id = ?", e.Id).Count(&count)
	if count > 0 {
		err = errors.New("存在子级菜单,不允许删除")
		return
	}
	// 2、校验是否存在用户
	if CheckMenuExistRole(e.Id) {
		err = errors.New("菜单已分配,不允许删除")
		return
	}
	if err = config.DB.Table(e.TableName()).Where("id = ?", e.Id).Delete(SysMenu{}).Error; err != nil {
		return
	}
	return
}

// 构建树结构
func (e *SysMenu) BuildTree(list []SysMenu, parentId string) []SysMenu {
	var tree []SysMenu
	for _, item := range list {
		if item.ParentId == parentId {
			children := e.BuildTree(list, item.Id)
			if len(children) > 0 {
				item.Children = children
			}
			tree = append(tree, item)
		}
	}
	return tree
}

// 构建前端所需要的路由菜单
func buildMenus(menus []SysMenu) []Router {
	routerList := []Router{}
	for _, menu := range menus {
		router := Router{Hidden: menu.Visible, Name: strings.Title(menu.Path), Path: getRouterPath(menu), Component: menu.Url}
		meta := Meta{Title: menu.Name, Icon: menu.Icon, ActiveMenu: menu.ActiveMenu}
		router.Meta = meta
		cMenus := menu.Children
		if len(cMenus) > 0 && menu.Type == "M" {
			router.AlwaysShow = true
			router.Redirect = "noRedirect"
			router.Children = buildMenus(cMenus)
		} else if menu.Type == "C" && menu.ParentId == "ROOT" {
			childrenList := []Router{}
			children := Router{Name: strings.Title(menu.Path), Path: menu.Path, Component: menu.Url}
			childMeta := Meta{Title: menu.Name, Icon: menu.Icon, ActiveMenu: menu.ActiveMenu}
			children.Meta = childMeta
			childrenList = append(childrenList, children)
			router.Children = childrenList
		}
		routerList = append(routerList, router)
	}
	return routerList
}

// 获取路由地址
func getRouterPath(menu SysMenu) string {
	routerPath := menu.Path
	if menu.Type == "M" {
		if menu.Url != config.PARENT_VIEW {
			routerPath = "/" + menu.Path
		}
	} else if menu.Type == "C" && menu.ParentId == "ROOT" {
		routerPath = "/"
	}
	return routerPath
}

// 改了角色与权限关联时，更新redis中的角色与权限关联
func UpdatePermByRoleId(roleId string) {
	perms := GetPermsMenuByRoleId(roleId)
	config.RedisConn.HSet(config.RolePermList, roleId, strings.Join(perms, ";"))
}

// 修改菜单，改了菜单的权限标识时，更新redis中的权限标识
func UpdatePerm(oldPerm, newPerm string) {
	redis := config.RedisConn
	result := make(map[string]interface{})
	data := redis.HGetAll(config.RolePermList).Val()
	for key, val := range data {
		split := strings.Split(val, ";")
		for i, s := range split {
			if s == oldPerm {
				split[i] = newPerm
			}
		}
		result[key] = strings.Join(split, ";")
	}
	redis.HMSet(config.RolePermList, result)
	redis.Expire(config.RolePermList, time.Second*604800)
}

// 删除角色时，删除redis中的角色与权限关联
func DeletePermByRoleId(roleIds []string) {
	for i := range roleIds {
		config.RedisConn.HDel(config.RolePermList, roleIds[i])
	}
}
