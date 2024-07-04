package sys

import "fiber-web-api/internal/app/common/config"

// 角色菜单关联
type SysRoleMenu struct {
	RoleId string `json:"roleId" form:"roleId"` // 角色ID
	MenuId string `json:"menuId" form:"menuId"` // 菜单ID
}

// 获取表名
func (SysRoleMenu) TableName() string {
	return "sys_role_menu"
}

// 新增角色和菜单关联
func (e *SysRoleMenu) Insert(menuIds []string) {
	// 先删除当前角色关联的菜单id
	config.DB.Table(e.TableName()).Where("role_id = ?", e.RoleId).Delete(SysRoleMenu{})
	// 再添加当前角色关联的菜单id
	var list []SysRoleMenu // 存放要添加的数据
	for _, menuId := range menuIds {
		item := SysRoleMenu{RoleId: e.RoleId, MenuId: menuId}
		list = append(list, item)
	}
	config.DB.Table(e.TableName()).Create(&list)
	UpdatePermByRoleId(e.RoleId)
}

// 删除角色和菜单关联
func (e *SysRoleMenu) Delete(roleIds []string) {
	// DELETE FROM `sys_role_menu` WHERE role_id in ('1','2','3')
	config.DB.Table(e.TableName()).Where("role_id in (?)", roleIds).Delete(SysRoleMenu{})
}

// 根据角色id获取菜单列表id
func (e *SysRoleMenu) GetMenuIdByRoleId() []string {
	var list []SysRoleMenu
	var result []string
	query := config.DB.Table(e.TableName())
	query.Where("role_id = ?", e.RoleId)
	query.Find(&list)
	for _, menu := range list {
		result = append(result, menu.MenuId)
	}
	return result
}

// 根据菜单id校验该菜单是否已分配给角色
func CheckMenuExistRole(menuId string) bool {
	var count int64
	config.DB.Table(SysRoleMenu{}.TableName()).Where("menu_id = ?", menuId).Count(&count)
	return count > 0
}

// 获取所有角色的权限标识
func GetPermsAll() map[string][]string {
	var list []SysRoleMenu
	sql := `
		select role_id,perms menu_id
        from sys_role_menu a
        left join sys_menu b on a.menu_id = b.id
	`
	config.DB.Table(SysRoleMenu{}.TableName()).Raw(sql).Find(&list)
	result := make(map[string][]string)
	// 根据RoleId进行分组
	for _, item := range list {
		result[item.RoleId] = append(result[item.RoleId], item.MenuId)
	}
	return result
}
