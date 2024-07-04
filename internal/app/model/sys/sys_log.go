package sys

import (
	"fiber-web-api/internal/app/common/config"
	"fmt"
	"github.com/google/uuid"
	"strings"
	"time"
)

// 操作日志管理
type SysLog struct {
	config.BaseModel
	IP     string `gorm:"ip" json:"ip"`         // 用户请求IP
	Title  string `gorm:"title" json:"title"`   // 用户请求的标题
	Type   string `gorm:"type" json:"title"`    // 操作类型（其他 登录 退出 新增 修改 删除 上传 导入 设置密码 重置密码）
	Method string `gorm:"method" json:"method"` // 用户请求的方法
	Url    string `gorm:"url" json:"url"`       // 请求url
	Info   string `gorm:"info" json:"info"`     // 详细信息
	State  string `gorm:"state" json:"state"`   // 状态（操作成功 操作失败）
}

// 获取表名
func (SysLog) TableName() string {
	return "sys_log"
}

// 列表
func (e *SysLog) GetPage(pageSize int, pageNum int) config.PageInfo {
	var list []SysLog // 查询结果
	var total int64   // 总数
	query := config.DB.Table(e.TableName())
	var creatorId string
	if e.CreatorId != nil {
		creatorId = *e.CreatorId
	}
	if creatorId != "" {
		query.Where("creator_id like ?", fmt.Sprintf("%%%s%%", creatorId))
	}
	if e.IP != "" {
		query.Where("ip like ?", fmt.Sprintf("%%%s%%", e.IP))
	}
	if !e.CreateTime.IsZero() {
		query.Where("DATE_FORMAT(create_time,'%Y-%m-%d') = ?", e.CreateTime.Format("2006-01-02"))
	}
	offset := (pageNum - 1) * pageSize                                                 // 计算跳过的记录数
	query.Debug().Order("create_time desc").Offset(offset).Limit(pageSize).Find(&list) // 分页查询，根据offset和limit来查询
	query.Count(&total)
	return config.PageInfo{list, total}
}

// 新增
func (e *SysLog) Insert() (err error) {
	e.Id = strings.ReplaceAll(uuid.NewString(), "-", "")
	e.CreateTime = time.Now()
	config.DB.Create(e)
	return
}
