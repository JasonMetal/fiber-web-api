package sys

import (
	"fiber-web-api/internal/app/common/config"
	"github.com/google/uuid"
	"strings"
)

// 安全中心
type SysSafe struct {
	config.BaseModel
	PwdCycle        int `json:"pwdCycle" form:"pwdCycle"`               // 密码更改周期（90天，60天，30天，0无）
	PwdLoginLimit   int `json:"pwdLoginLimit" form:"pwdLoginLimit"`     // 密码登录限制（0：连续错3次，锁定账号15分钟。1：连续错5次，锁定账号30分钟）
	IdleTimeSetting int `json:"idleTimeSetting" form:"idleTimeSetting"` // 闲置时间设置（0：无。1：空闲30分钟，系统默认用户退出）
}

// 获取表名
func (SysSafe) TableName() string {
	return "sys_safe"
}

// 详情
func (e *SysSafe) GetById() (err error) {
	query := config.DB.Table(e.TableName())
	if err = query.First(e).Error; err != nil {
		return
	}
	return
}

// 修改
func (e *SysSafe) Update() (err error) {
	if e.Id == "" {
		e.Id = strings.ReplaceAll(uuid.NewString(), "-", "")
		e.CreatorId = GetLoginId(e.Token)
		config.DB.Create(e)
	} else {
		// 使用Save方法进行更新，标识零值也需要进行更新。Select是指定需要更新哪些字段
		config.DB.Model(&SysSafe{}).Select("pwd_cycle", "pwd_login_limit", "idle_time_setting").Where("id = ?", e.Id).Save(e)
		expire := GetTimeOut(e.Token)
		i := e.IdleTimeSetting
		//修改token的过期时间
		if expire > 0 && i == 0 {
			UpdateTimeOut(e.Token, -1)
		} else if expire < 0 && i != 0 {
			UpdateTimeOut(e.Token, config.TokenExpire)
		}
	}
	return
}
