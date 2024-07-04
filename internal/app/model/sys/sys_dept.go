package sys

import (
	"fiber-web-api/internal/app/common/config"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"strings"
	"time"
)

// 部门管理
type SysDept struct {
	config.BaseModel
	Name     string    `json:"name" form:"name"`         // 名称
	ParentId string    `json:"parentId" form:"parentId"` // 上级部门id
	Level    int       `json:"level" form:"level"`       // 层级（1 根目录 2 单位 3 部门 4 小组）
	Sort     int       `json:"sort" form:"sort"`         // 序号
	Children []SysDept `gorm:"-" json:"children"`        // 子级数据
}

// 获取表名
func (SysDept) TableName() string {
	return "sys_dept"
}

// 递归所有子节点
func (e *SysDept) ChildList() []SysDept {
	// 获取所以子节点包括它本身
	sql := `
		WITH RECURSIVE temp_dept AS (
            SELECT id,name,parent_id FROM sys_dept WHERE id = ?
            UNION ALL
            SELECT d.id,d.name,d.parent_id FROM sys_dept d
            JOIN temp_dept sd ON sd.id = d.parent_id
        )
        SELECT * FROM temp_dept
	`
	var childList []SysDept
	config.DB.Raw(sql, e.ParentId).Find(&childList)
	return childList
}

// 根据parentId得到所有子节点的id和name（包括它本身），以逗号分隔
func GetDeptChild(parentId string) (string, string) {
	dept := SysDept{}
	dept.ParentId = parentId
	list := dept.ChildList()
	if len(list) > 0 {
		idList := []string{}
		nameList := []string{}
		for _, t := range list {
			idList = append(idList, t.Id)
			nameList = append(nameList, t.Name)
		}
		return strings.Join(idList, ","), strings.Join(nameList, ",")
	}
	return "", ""
}

// 递归所有父节点，获取获取祖级列表（不包括它本身）
func (e *SysDept) GetAncestor() (string, string) {
	sql := `
		WITH RECURSIVE temp_dept(id,name,parent_id,level) AS (
			SELECT id,name,parent_id,level
			FROM sys_dept
			WHERE id = ?
			UNION ALL
			SELECT d.id,d.name,d.parent_id,d.level
			FROM sys_dept d
			JOIN temp_dept c ON d.id = c.parent_id
		)
		SELECT id,name,parent_id,level
		FROM temp_dept
		WHERE id != ?
		ORDER BY level,parent_id
	`
	var ancestorList []SysDept
	config.DB.Raw(sql, e.Id, e.Id).Find(&ancestorList)
	idList := []string{}
	nameList := []string{}
	for _, t := range ancestorList {
		idList = append(idList, t.Id)
		nameList = append(nameList, t.Name)
	}
	return strings.Join(idList, ","), strings.Join(nameList, ",")
}

// 树形列表
func (e *SysDept) GetListTree() []SysDept {
	var list []SysDept // 查询结果
	sql := ""
	args := []interface{}{}
	loginUser := GetLoginUser(e.Token)
	e.Id = loginUser.DeptId
	if e.Id != "" {
		// 这里用于递归 e.Id 的父级根节点，因为转为树结构时，是从根节点开始递归的（不包括e.Id本身）
		sql = `WITH RECURSIVE temp_dept(id,parent_id,name,level,sort) AS (
                SELECT id,parent_id,name,level,sort
                FROM sys_dept
                WHERE id = ?
                UNION ALL
                SELECT d.id,d.parent_id,d.name,d.level,d.sort
                FROM sys_dept d
                JOIN temp_dept c ON d.id = c.parent_id
            )
            SELECT id,parent_id,name,level,sort
            FROM temp_dept
            WHERE id != ?
            UNION ALL`
		args = append(args, e.Id, e.Id)
	}
	sql += ` SELECT id,parent_id,name,level,sort FROM sys_dept`
	// 设置数据范围查询
	scope := AppendQueryDataScope(e.Token, "id", "2", false, true)
	if scope != "" {
		sql = sql + " WHERE " + scope
	}
	config.DB.Table(e.TableName()).Debug().Order("`level`,parent_id,sort asc").Raw(sql, args...).Find(&list)
	return e.BuildTree(list, "ROOT")
}

// 获取详情
func (e *SysDept) GetById() (err error) {
	if !CheckDataScope(e.Token, e.Id, false, true) {
		err = errors.New("没有操作权限！")
		return
	}
	config.DB.Table(e.TableName()).Where("id = ?", e.Id).Find(e)
	return
}

// 新增
func (e *SysDept) Insert() (err error) {
	// 新增部门时，只允许新增子部门（也就是只允许给当前用户所在部门新增子部门）
	if !CheckDataScope(e.Token, e.ParentId, false, true) {
		err = errors.New("没有操作权限！")
		return
	}
	// 校验用户名和手机号码
	var count int64
	query := config.DB.Table(e.TableName())
	query.Where("name = ? and parent_id = ?", e.Name, e.ParentId).Count(&count)
	if count > 0 {
		err = errors.New("名称已存在！")
		return
	}
	err = e.getLevel()
	if err != nil {
		return err
	}
	e.Id = strings.ReplaceAll(uuid.NewString(), "-", "")
	e.CreatorId = GetLoginId(e.Token)
	e.CreateTime = time.Now()
	config.DB.Table(e.TableName()).Create(e)
	// 新增成功，更新数据权限缓存
	exists, _ := config.RedisConn.Exists(config.DATA_SCOPE + e.ParentId).Result()
	if exists > 0 {
		childId := config.RedisConn.HGet(config.DATA_SCOPE+e.ParentId, "childId").Val()
		childName := config.RedisConn.HGet(config.DATA_SCOPE+e.ParentId, "childName").Val()
		config.RedisConn.HSet(config.DATA_SCOPE+e.ParentId, "childId", childId+","+e.Id)
		config.RedisConn.HSet(config.DATA_SCOPE+e.ParentId, "childName", childName+","+e.Name)
	}
	return
}

// 修改
func (e *SysDept) Update() (err error) {
	// 修改部门时，只允许修改当前部门和子部门数据
	if !CheckDataScope(e.Token, e.Id, false, true) {
		err = errors.New("没有操作权限！")
		return
	}
	// 校验用户名和手机号码
	var count int64
	query := config.DB.Table(e.TableName())
	query.Where("name = ? and parent_id = ? and id <> ?", e.Name, e.ParentId, e.Id).Count(&count)
	if count > 0 {
		err = errors.New("名称已存在！")
		return
	}
	err = e.getLevel()
	if err != nil {
		return err
	}
	config.DB.Table(e.TableName()).Model(&SysDept{}).Where("id = ?", e.Id).Updates(e)
	return
}

// 删除
func (e *SysDept) Delete() (err error) {
	// 修改部门时，只允许修改当前部门和子部门数据
	if !CheckDataScope(e.Token, e.Id, false, true) {
		err = errors.New("没有操作权限！")
		return
	}
	// 1、校验是否存在下级
	var count int64
	query := config.DB.Table(e.TableName())
	query.Where("parent_id = ?", e.Id).Count(&count)
	if count > 0 {
		err = errors.New("存在下级,不允许删除")
		return
	}
	// 2、校验是否存在用户
	if CheckDeptExistUser(e.Id) {
		err = errors.New("该组织存在用户,不允许删除")
		return
	}
	if err = config.DB.Table(e.TableName()).Delete(e).Error; err != nil {
		return
	}
	return
}

// 新增或修改部门时，根据父级的level获取当前部门的level
func (e *SysDept) getLevel() (err error) {
	if e.ParentId == "ROOT" {
		e.Level = 1
	} else {
		var parent SysDept
		parent.Id = e.ParentId
		parent.GetById()
		if parent.Name == "" {
			err = errors.New("上级不存在！")
			return
		}
		e.Level = parent.Level + 1
	}
	return
}

// 构建树结构
func (e *SysDept) BuildTree(list []SysDept, parentId string) []SysDept {
	var tree []SysDept
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
