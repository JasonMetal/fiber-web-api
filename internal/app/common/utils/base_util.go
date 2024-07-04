package utils

import (
	"fiber-web-api/internal/app/common/config"
	"github.com/mojocn/base64Captcha"
	"github.com/mozillazg/go-pinyin"
	"github.com/pkg/errors"
	"image/color"
	"io"
	mrand "math/rand"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// 验证码，第一个参数是验证码的上限个数（最多存多少个验证码），第二个参数是验证码的有效时间
var captchaStore = base64Captcha.NewMemoryStore(500, 1*time.Minute)

// 生成验证码
func GenerateCaptcha(length, width, height int) (lid string, lb64s string) {
	var driver base64Captcha.Driver
	var driverString base64Captcha.DriverString
	// 配置验证码信息
	captchaConfig := base64Captcha.DriverString{
		Height:          height,
		Width:           width,
		NoiseCount:      0,
		ShowLineOptions: 2 | 4,
		Length:          length,
		Source:          config.RandomCaptcha,
		BgColor: &color.RGBA{
			R: 3,
			G: 102,
			B: 214,
			A: 125,
		},
		Fonts: []string{"wqy-microhei.ttc"},
	}
	driverString = captchaConfig
	driver = driverString.ConvertFonts()
	captcha := base64Captcha.NewCaptcha(driver, captchaStore)
	lid, lb64s, _, _ = captcha.Generate()
	//fmt.Println(lid)
	//fmt.Println(lb64s)
	return
}

// 生成随机字符串作为令牌
func GenerateRandomToken(length int) string {
	tokenBytes := make([]byte, length)
	mrand.NewSource(time.Now().UnixNano())
	for i := range tokenBytes {
		tokenBytes[i] = config.RandomCharset[mrand.Intn(len(config.RandomCharset))]
	}
	return string(tokenBytes)
}

// 判断数组中是否包含指定元素
func IsContain(items interface{}, item interface{}) bool {
	switch items.(type) {
	case []int:
		intArr := items.([]int)
		for _, value := range intArr {
			if value == item.(int) {
				return true
			}
		}
	case []string:
		strArr := items.([]string)
		for _, value := range strArr {
			if value == item.(string) {
				return true
			}
		}
	default:
		return false
	}
	return false
}

// 中文转拼音大写（首字母）
func ConvertToPinyin(text string, p pinyin.Args) string {
	var initials []string
	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fff { // 判断字符是否为中文字符
			pinyinResult := pinyin.Pinyin(string(r), p)
			initials = append(initials, strings.ToUpper(string(pinyinResult[0][0][0])))
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9' || r == '_') { // 判断字符是否为英文字母或数字
			initials = append(initials, strings.ToUpper(string(r)))
		}
	}
	return strings.Join(initials, "")
}

// 上传保存文件 form 文件数据、relative 文件保存的相对路径、fileName 文件名称
func SaveFile(form *multipart.Form, relative, fileName string) error {
	// 获取文件数据
	file, err := form.File["file"][0].Open()
	if err != nil {
		err = errors.New("上传文件失败：" + err.Error())
		return err
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		err = errors.New("上传文件失败：" + err.Error())
		return err
	}
	// 获取当前项目路径
	currentDir, err := os.Getwd()
	if err != nil {
		err = errors.New("获取项目路径失败：" + err.Error())
		return err
	}
	// 文件上传的绝对路径
	absolute := filepath.Join(currentDir, relative)
	// 创建目录
	err = os.MkdirAll(absolute, os.ModePerm)
	if err != nil {
		err = errors.New("上传文件失败：" + err.Error())
		return err
	}
	// 在当前项目路径中的 /upload/20231208/ 下创建新文件
	newFile, err := os.Create(filepath.Join(absolute, fileName))
	if err != nil {
		err = errors.New("上传文件失败：" + err.Error())
		return err
	}
	defer newFile.Close()
	// 将文件内容写入新文件
	_, err = newFile.Write(content)
	if err != nil {
		err = errors.New("上传文件失败：" + err.Error())
		return err
	}
	return nil
}
