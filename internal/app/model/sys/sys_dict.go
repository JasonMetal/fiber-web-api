package sys

import (
	"fiber-web-api/internal/app/common/config"
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"strconv"
	"strings"
	"time"
)

// 字典管理
type SysDict struct {
	config.BaseModel
	ParentId  string    `json:"parentId" form:"parentId"`   // 上级id
	DictName  string    `json:"dictName" form:"dictName"`   // 字典名称
	DictCode  string    `json:"dictCode" form:"dictCode"`   // 字典代码
	DictValue string    `json:"dictValue" form:"dictValue"` // 字典值
	Sort      int       `json:"sort" form:"sort"`           // 排序
	IsType    int       `json:"isType" form:"isType"`       // 是否是字典类型（1 字典类型 2 字典项）
	Remark    string    `json:"remark" form:"remark"`       // 备注
	Children  []SysDict `gorm:"-" json:"children"`          // 子级数据
}

// 获取表名
func (SysDict) TableName() string {
	return "sys_dict"
}

// 字典类型列表
func (e *SysDict) GetTypeList() []SysDict {
	var list []SysDict
	query := config.DB.Table(e.TableName())
	query.Where("is_type = 1")
	if e.DictName != "" {
		query.Where("dict_name like ?", fmt.Sprintf("%%%s%%", e.DictName))
	}
	query.Debug().Order("parent_id,sort asc").Find(&list)
	return buildDictTree(list, "ROOT")
}

// 列表
func (e *SysDict) GetPage(pageSize int, pageNum int) config.PageInfo {
	var list []SysDict // 查询结果
	var total int64    // 总数
	query := config.DB.Table(e.TableName())
	query.Where("is_type = 2")
	if e.DictName != "" {
		query.Where("dict_name like ?", fmt.Sprintf("%%%s%%", e.DictName))
	}
	if e.DictCode != "" {
		query.Where("dict_code like ?", fmt.Sprintf("%%%s%%", e.DictCode))
	}
	if e.ParentId != "" {
		query.Where("parent_id = ?", e.ParentId)
	}
	offset := (pageNum - 1) * pageSize                                                   // 计算跳过的记录数
	query.Debug().Order("parent_id,sort asc").Offset(offset).Limit(pageSize).Find(&list) // 分页查询，根据offset和limit来查询
	query.Count(&total)
	return config.PageInfo{list, total}
}

// 获取详情
func (e *SysDict) GetById() {
	config.DB.Table(e.TableName()).Where("id = ?", e.Id).Find(e)
}

// 详情
func (e *SysDict) HasDictByNameAndCode() bool {
	var count int64
	query := config.DB.Table(e.TableName())
	if e.Id != "" {
		query.Where("id <> ?", e.Id)
	}
	if e.DictName != "" {
		query.Where("dict_name = ?", e.DictName)
	}
	if e.DictCode != "" {
		query.Where("dict_code = ?", e.DictCode)
	}
	query.Count(&count)
	return count > 0
}

// 生成字典名称或字典代码
func (e *SysDict) CreateNameOrCode() string {
	index := -1
	str := ""
	if e.DictName != "" {
		str = e.DictName
	}
	if e.DictCode != "" {
		str = e.DictCode
	}
	index = strings.LastIndex(str, "_")
	if index != -1 {
		str = str[:index] // 截取从索引0到索引index的子串（包括索引0，不包括索引index）
	}
	return e.rightLikeNameOrCode(str)
}

// 字典名称或代码右模糊匹配
func (e *SysDict) rightLikeNameOrCode(str string) string {
	var list []SysDict
	query := config.DB.Table(SysDict{}.TableName())
	if e.DictName != "" {
		query.Where("dict_name LIKE ?", fmt.Sprintf("%s%%", str)) // 右模糊查询
	}
	if e.DictCode != "" {
		query.Where("dict_code LIKE ?", fmt.Sprintf("%s%%", str)) // 右模糊查询
	}
	query.Debug().Order("create_time desc").Find(&list) // 根据创建时间倒序，查询最新的那一个
	if len(list) > 0 {
		dict := ""
		if e.DictName != "" {
			dict = list[0].DictName
		}
		if e.DictCode != "" {
			dict = list[0].DictCode
		}
		strIndex := strings.LastIndex(dict, "_")
		if strIndex != -1 {
			dict = dict[strIndex+1:] // 截取从索引strIndex+1到字符串末尾的子串（包括索引strIndex+1）
			i, err := strconv.ParseInt(dict, 10, 64)
			if err == nil {
				str = str + "_" + fmt.Sprintf("%d", (i+1))
			} else {
				str = str + "_1"
			}
		} else {
			str = str + "_1"
		}
	}
	return str
}

// 新增
func (e *SysDict) Insert() (err error) {
	query := config.DB.Table(e.TableName())
	// 如果字典名称已存在，不提示重复，直接生成新的字典名称
	if checkDictNameAndCode(e.DictName, "", "") {
		dict := SysDict{}
		dict.DictName = e.DictName
		name := dict.CreateNameOrCode()
		e.DictName = name
	}
	// 如果字典代码已存在，不提示重复，直接生成新的字典代码
	if checkDictNameAndCode("", e.DictCode, "") {
		dict := SysDict{}
		dict.DictCode = e.DictCode
		code := dict.CreateNameOrCode()
		e.DictCode = code
	}
	if e.ParentId == "0" {
		e.ParentId = "ROOT"
	}
	e.Id = strings.ReplaceAll(uuid.NewString(), "-", "")
	e.CreateTime = time.Now()
	query.Create(e)
	return
}

// 修改
func (e *SysDict) Update() (err error) {
	// 如果字典名称已存在，不提示重复，直接生成新的字典名称
	if checkDictNameAndCode(e.DictName, "", e.Id) {
		dict := SysDict{}
		dict.DictName = e.DictName
		name := dict.CreateNameOrCode()
		e.DictName = name
	}
	// 如果字典代码已存在，不提示重复，直接生成新的字典代码
	if checkDictNameAndCode("", e.DictCode, e.Id) {
		dict := SysDict{}
		dict.DictCode = e.DictCode
		code := dict.CreateNameOrCode()
		e.DictCode = code
	}
	config.DB.Model(&SysDict{}).Omit("id", "create_time").Where("id = ?", e.Id).Save(e)
	return
}

// 删除字典类型
func (e *SysDict) DeleteType() (err error) {
	var count int64
	query := config.DB.Table(e.TableName())
	query.Where("parent_id = ?", e.Id).Count(&count)
	if count > 0 {
		err = errors.New("存在子级,不允许删除")
		return
	}
	config.DB.Table(e.TableName()).Where("id = ?", e.Id).Delete(SysDict{})
	return
}

// 删除字典项
func (e *SysDict) Delete(ids []string) (err error) {
	config.DB.Table(e.TableName()).Delete(&SysRole{}, ids)
	return
}

// 角色下拉列表
func (e *SysDict) GetSelectList() []SysDict {
	var dict SysDict
	var list []SysDict // 查询结果
	// 先根据字典代码查询字典类型
	config.DB.Table(e.TableName()).Where("dict_code = ?", e.DictCode).Find(&dict)
	// 再根据字典类型的id查询它下面的字典项列表
	//config.DB.Table(e.TableName()).Where("parent_id = ? and is_type = 2", dict.Id).Debug().Order("sort asc").Find(&list)
	config.DB.Table(e.TableName()).Where("parent_id = ?", dict.Id).Debug().Order("sort asc").Find(&list)
	return list
}

// 构建树结构
func buildDictTree(list []SysDict, parentId string) []SysDict {
	var tree []SysDict
	for _, item := range list {
		if item.ParentId == parentId {
			children := buildDictTree(list, item.Id)
			if len(children) > 0 {
				item.Children = children
			}
			tree = append(tree, item)
		}
	}
	return tree
}

// 校验字典名称和代码是否存在
func checkDictNameAndCode(roleName, roleKey, id string) bool {
	var count int64
	query := config.DB.Table(SysDict{}.TableName())
	if roleName != "" {
		query.Where("dict_name = ?", roleName)
	}
	if roleKey != "" {
		query.Where("dict_code = ?", roleKey)
	}
	if id != "" {
		query.Where("id <> ?", id)
	}
	query.Count(&count)
	return count > 0
}
