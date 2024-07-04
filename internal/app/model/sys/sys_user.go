package sys

import (
	"errors"
	"fiber-web-api/internal/app/common/config"
	"fiber-web-api/internal/app/common/utils"
	"fmt"
	"github.com/google/uuid"
	"strings"
	"time"
)

// 用户信息model，对应数据库的 sys_user 表
type SysUser struct {
	config.BaseModel // 嵌套公共的model，这样就可以使用 BaseModel 的字段了
	SysUserView
	Password string `gorm:"password" json:"password" form:"password"` // 加密密码
}

// 用户信息model，用于展示给前端
type SysUserView struct {
	config.BaseModel         // 嵌套公共的model，这样就可以使用 BaseModel 的字段了
	UserName         string  `json:"userName" form:"userName"`         // 用户名称
	RealName         string  `json:"realName" form:"realName"`         // 真实姓名
	DeptId           string  `json:"deptId" form:"deptId"`             // 部门id
	DeptName         string  `json:"deptName" form:"deptName"`         // 部门名称
	AncestorId       string  `json:"ancestorId" form:"ancestorId"`     // 祖级id
	AncestorName     string  `json:"ancestorName" form:"ancestorName"` // 祖级名称
	ChildId          string  `json:"childId" form:"childId"`           // 部门id及子级id
	ChildName        string  `json:"childName" form:"childName"`       // 部门名称及子级名称
	RoleId           string  `json:"roleId" form:"roleId"`             // 角色id
	RoleKey          string  `json:"roleKey" form:"roleKey"`           // 角色代码
	RoleName         string  `json:"roleName" form:"roleName"`         // 角色名称
	Phone            *string `json:"phone" form:"phone"`               // 联系电话 这里用指针，是因为可以传空（这个空不是指空字符串，而是null）
	State            int     `json:"state" form:"state"`               // 状态（1 启用 2 停用）
	Picture          *string `json:"picture" form:"picture"`           // 头像地址
}

// 密码结构体，用于修改密码
type Password struct {
	config.BaseModel        // 嵌套公共的model，这样就可以使用 BaseModel 的字段了
	OldPassword      string `json:"oldPassword" form:"oldPassword"` // 旧密码
	NewPassword      string `json:"newPassword" form:"newPassword"` // 新密码
}

// 新增、更新用户信息时，要忽略的字段
var omit = "dept_name,ancestor_id,ancestor_name,child_id,child_name,role_key,role_name"

// 获取用户管理的表名
func (SysUserView) TableName() string {
	return "sys_user"
}

// 列表
func (e *SysUserView) GetPage(pageSize int, pageNum int) config.PageInfo {
	var list []SysUserView // 查询结果
	var total int64        // 总数
	query := config.DB.Table(e.TableName())
	if e.UserName != "" {
		query.Where("user_name like ?", fmt.Sprintf("%%%s%%", e.UserName))
	}
	if e.RealName != "" {
		query.Where("real_name like ?", fmt.Sprintf("%%%s%%", e.RealName))
	}
	if e.AncestorId != "" {
		ids, _ := GetDeptChild(e.AncestorId) // 获取当前 parentId 的所有子节点包括它本身
		query.Where("FIND_IN_SET(dept_id,?)", ids)
	}
	// 数据过滤
	scope := AppendQueryDataScope(e.Token, "dept_id", "2", false, true)
	if scope != "" {
		query.Where(scope)
	}
	// 关联部门和角色表查询
	offset := (pageNum - 1) * pageSize // 计算跳过的记录数
	sql := "sys_user.id,user_name,real_name,dept_id,role_id,CONCAT(SUBSTRING(phone, 1, 3), REPEAT('*', LENGTH(phone) - 3)) phone,sys_user.state,picture,b.name as dept_name,role_key,role_name"
	query.Debug().Order("sys_user.create_time desc").Select(sql).
		Joins("left join sys_dept b on b.id = sys_user.dept_id").
		Joins("left join sys_role c on c.id = sys_user.role_id").
		Offset(offset).Limit(pageSize).Find(&list) // 分页查询，根据offset和limit来查询
	query.Count(&total) // 查询总数用Count，注意查总数不能直接连在Find()后面，需要分开来单独一句才能正确查询到总数。
	return config.PageInfo{list, total}
}

// 详情
func (e *SysUser) GetUser() (err error) {
	query := config.DB.Table(e.TableName())
	sql := `
		SELECT a.*,b.name dept_name,role_key,role_name
		FROM sys_user a
		LEFT JOIN sys_dept b on FIND_IN_SET(b.id,a.dept_id)
		LEFT JOIN sys_role c on a.role_id=c.id
	`
	args := []interface{}{}
	if e.Id != "" {
		sql = sql + "WHERE a.id = ?"
		args = append(args, e.Id)
	}
	if e.UserName != "" {
		sql = sql + "WHERE user_name = ?"
		args = append(args, e.UserName)
	}
	// 数据过滤
	scope := AppendQueryDataScope(e.Token, "dept_id", "2", false, true)
	if scope != "" {
		sql = sql + " AND " + scope
	}
	if err = query.Raw(sql, args...).Find(e).Error; err != nil {
		return
	}
	if e.Id == "" || e.UserName == "" {
		err = errors.New("没有查看权限！")
		return
	}
	return
}

// 新增
func (e *SysUser) Insert() (err error) {
	// 校验新增的用户和当前用户是否是同一部门或子部门
	if !CheckDataScope(e.Token, e.DeptId, false, true) {
		err = errors.New("没有操作权限！")
		return
	}
	// 校验用户名和手机号码
	var count int64
	db := config.DB.Table(e.TableName())
	db.Where("user_name = ?", e.UserName).Count(&count)
	if count > 0 {
		err = errors.New("用户名称已存在！")
		return
	}
	if e.Phone != nil {
		//phone := utils.RSADecrypt(*e.Phone);   // 手机号码私钥解密
		//e.Phone = &phone
		db.Where("phone = ?", e.Phone).Count(&count)
		if count > 0 {
			err = errors.New("手机号码已存在！")
			return
		}
	}
	e.Id = strings.ReplaceAll(uuid.NewString(), "-", "")
	e.CreatorId = GetLoginId(e.Token)
	e.CreateTime = time.Now()
	config.DB.Table(e.TableName()).Omit(omit).Create(e)
	return
}

// 修改
func (e *SysUser) Update() (err error) {
	// 校验修改的用户和当前用户是否是同一部门或子部门
	if !CheckDataScope(e.Token, e.DeptId, false, true) {
		err = errors.New("没有操作权限！")
		return
	}
	var byId SysUser
	config.DB.Table(e.TableName()).Where("id = ?", e.Id).Find(&byId)
	if byId.DeptId != e.DeptId && !CheckDataScope(e.Token, byId.DeptId, false, true) {
		err = errors.New("没有操作权限！")
		return
	}
	// 校验用户名和手机号码
	var count int64
	db := config.DB.Table(e.TableName())
	db.Where("user_name = ? and id <> ?", e.UserName, e.Id).Count(&count)
	if count > 0 {
		err = errors.New("用户名称已存在！")
		return
	}
	if e.Phone != nil {
		//phone := utils.RSADecrypt(*e.Phone);   // 手机号码私钥解密
		//e.Phone = &phone
		db.Where("phone = ? and id <> ?", e.Phone, e.Id).Count(&count)
		if count > 0 {
			err = errors.New("手机号码已存在！")
			return
		}
	}
	config.DB.Table(e.TableName()).Omit(omit).Model(&SysUser{}).Where("id = ?", e.Id).Updates(e)
	return
}

// 删除
func (e *SysUser) Delete(ids []string) (err error) {
	// 先查询要删除的用户
	scope := GetDataScope(e.Token, false, true)
	if scope != "" {
		var list []SysUser
		config.DB.Table(e.TableName()).Where("id in (?)", ids).Find(&list)
		split := strings.Split(scope, ",")
		for _, user := range list {
			if !utils.IsContain(split, user.DeptId) {
				err = errors.New("没有操作权限！")
				return
			}
		}
	}
	config.DB.Table(e.TableName()).Delete(&SysUser{}, ids)
	return
}

// 修改密码
func (e *Password) UpdatePassword() (err error) {
	if e.NewPassword == "" || e.OldPassword == "" || e.Id == "" {
		err = errors.New("数据解密失败")
		return
	}
	var user SysUser
	config.DB.Table(user.TableName()).Where("id = ?", e.Id).Find(&user)
	if !CheckDataScope(e.Token, user.DeptId, false, true) {
		err = errors.New("没有操作权限！")
		return
	}
	if e.NewPassword == e.OldPassword {
		err = errors.New("新密码不可于旧密码相同")
		return
	}
	if e.NewPassword == config.InitPassword {
		err = errors.New("新密码不可于初始密码相同")
		return
	}
	// 正则表达式我去你大爷！！！不校验了，我校验你大爷！！！从别的项目复制过来的正则，为什么你就死活校验不上！！
	// 同样的正则，我一模一样复制到校验工具去验证都可以匹配上，就你不行，我去你大爷，不校验了，校验你爹！！！
	/*reg := "^(?=.*?[A-Z])(?=.*?[a-z])(?=.*?[0-9])(?=.*?[_#?!@$%^&*-]).{8,}$"
	m, _ := regexp.MatchString(reg, e.NewPassword)
	if !m {
		err = errors.New("密码长度大于7,且必须由数字、大小写字母、特殊字符组成")
		return
	}*/
	b := utils.AuthenticatePassword(e.OldPassword, user.Password)
	if !b {
		err = errors.New("旧密码错误")
		return
	}
	newPassword, err := utils.GetEncryptedPassword(e.NewPassword)
	if err = config.DB.Table(user.TableName()).Where("id = ?", e.Id).Update("password", newPassword).Error; err != nil {
		err = errors.New("密码修改失败")
		return
	}
	return
}

// 重置密码为初始密码
func (e *SysUser) ResetPassword() (err error) {
	var user SysUser
	config.DB.Table(e.TableName()).Where("id = ?", e.Id).Find(&user)
	if !CheckDataScope(e.Token, user.DeptId, false, true) {
		err = errors.New("没有操作权限！")
		return
	}
	password, err := utils.GetEncryptedPassword(config.InitPassword)
	if err = config.DB.Table(e.TableName()).Where("id = ?", e.Id).Update("password", password).Error; err != nil {
		err = errors.New("密码重置失败")
		return
	}
	return
}

// 上传头像
func (e *SysUser) Upload() {
	id := GetLoginId(e.Token)
	config.DB.Table(e.TableName()).Where("id = ?", id).Update("picture", e.Picture)
}

// 根据部门id校验是否存在用户
func CheckDeptExistUser(deptId string) bool {
	var count int64
	query := config.DB.Table(SysUserView{}.TableName())
	query.Where("dept_id = ?", deptId).Count(&count)
	return count > 0
}

// 根据角色id校验是否存在用户
func CheckRoleExistUser(roleId string) bool {
	var count int64
	query := config.DB.Table(SysUserView{}.TableName())
	query.Where("role_id = ?", roleId).Count(&count)
	return count > 0
}
