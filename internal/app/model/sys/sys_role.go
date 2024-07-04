package sys

import (
	"fiber-web-api/internal/app/common/config"
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"strings"
	"time"
)

// 角色管理
type SysRole struct {
	config.BaseModel
	RoleKey  string   `json:"roleKey" form:"roleKey"`          // 角色代码
	RoleName string   `json:"roleName" form:"roleName"`        // 角色名称
	IsOpen   bool     `json:"isOpen" form:"isOpen"`            // 菜单树是否展开（0折叠 1展开 ）
	State    int      `json:"state" form:"state"`              // 角色状态（1正常 2停用 3删除）
	Remark   string   `json:"remark" form:"remark"`            // 备注
	MenuIds  []string `gorm:"-" json:"menuIds" form:"menuIds"` // 菜单组
}

// 获取表名
func (SysRole) TableName() string {
	return "sys_role"
}

// 列表
func (e *SysRole) GetPage(pageSize int, pageNum int) config.PageInfo {
	var list []SysRole // 查询结果
	var total int64    // 总数
	query := config.DB.Table(e.TableName())
	if e.RoleName != "" {
		query.Where("role_name like ?", fmt.Sprintf("%%%s%%", e.RoleName))
	}
	if e.RoleKey != "" {
		query.Where("role_key like ?", fmt.Sprintf("%%%s%%", e.RoleKey))
	}
	offset := (pageNum - 1) * pageSize                                                 // 计算跳过的记录数
	query.Debug().Order("create_time desc").Offset(offset).Limit(pageSize).Find(&list) // 分页查询，根据offset和limit来查询
	query.Count(&total)
	return config.PageInfo{list, total}
}

// 详情
func (e *SysRole) GetById() {
	config.DB.Table(e.TableName()).Where("id = ?", e.Id).Find(e)
}

// 新增
func (e *SysRole) Insert() (err error) {
	// 校验角色名称和角色代码
	if checkRoleNameAndKey(e.RoleName, "", "") {
		err = errors.New("角色名称已存在！")
		return
	}
	if checkRoleNameAndKey("", e.RoleKey, "") {
		err = errors.New("角色代码已存在！")
		return
	}
	e.Id = strings.ReplaceAll(uuid.NewString(), "-", "")
	e.CreatorId = GetLoginId(e.Token)
	e.CreateTime = time.Now()
	config.DB.Table(e.TableName()).Create(e)
	// 保存菜单树
	roleMenu := SysRoleMenu{RoleId: e.Id}
	roleMenu.Insert(e.MenuIds)
	return
}

// 修改
func (e *SysRole) Update() (err error) {
	// 校验角色名称和角色代码
	if checkRoleNameAndKey(e.RoleName, "", e.Id) {
		err = errors.New("角色名称已存在！")
		return
	}
	if checkRoleNameAndKey("", e.RoleKey, e.Id) {
		err = errors.New("角色代码已存在！")
		return
	}
	config.DB.Model(&SysRole{}).Select("role_key", "role_name", "is_open", "state", "remark").Where("id = ?", e.Id).Save(e)
	// 保存菜单树
	roleMenu := SysRoleMenu{RoleId: e.Id}
	roleMenu.Insert(e.MenuIds)
	return
}

// 修改状态
func (e *SysRole) UpdateState() (err error) {
	config.DB.Model(&SysRole{}).Select("state").Where("id = ?", e.Id).Save(e)
	return
}

// 删除
func (e *SysRole) Delete(ids []string) (err error) {
	for _, id := range ids {
		e.Id = id
		e.GetById()
		// 首先查询角色是否已分配用户
		if CheckRoleExistUser(id) {
			err = errors.New(fmt.Sprintf("%s角色已分配，不允许删除", e.RoleName))
			return
		}
	}
	if err = config.DB.Table(e.TableName()).Delete(&SysRole{}, ids).Error; err != nil {
		return
	}
	// 删除角色同时删除角色菜单关联
	roleMenu := &SysRoleMenu{}
	roleMenu.Delete(ids)
	DeletePermByRoleId(ids) // 删除对应角色的权限标识缓存
	return
}

// 角色下拉列表
func (e *SysRole) GetSelectList() []SysRole {
	var list []SysRole // 查询结果
	config.DB.Table(e.TableName()).Find(&list)
	return list
}

// 校验角色名称和代码是否存在
func checkRoleNameAndKey(roleName, roleKey, id string) bool {
	var count int64
	query := config.DB.Table(SysRole{}.TableName())
	if roleName != "" {
		query.Where("role_name = ?", roleName)
	}
	if roleKey != "" {
		query.Where("role_key = ?", roleKey)
	}
	if id != "" {
		query.Where("id <> ?", id)
	}
	query.Count(&count)
	return count > 0
}
