package mylog

import (
	"fmt"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func Info(msg string) {
	logOut(msg, "Info")
}

func Debug(msg string) {
	logOut(msg, "Debug")
}

func Error(msg string) {
	logOut(msg, "Error")
}

func LogOut(msg string) {
	// 替换掉彩色打印符号
	msg = strings.ReplaceAll(msg, logger.Reset, "")
	msg = strings.ReplaceAll(msg, logger.Red, "")
	msg = strings.ReplaceAll(msg, logger.Green, "")
	msg = strings.ReplaceAll(msg, logger.Yellow, "")
	msg = strings.ReplaceAll(msg, logger.Blue, "")
	msg = strings.ReplaceAll(msg, logger.Magenta, "")
	msg = strings.ReplaceAll(msg, logger.Cyan, "")
	msg = strings.ReplaceAll(msg, logger.White, "")
	msg = strings.ReplaceAll(msg, logger.BlueBold, "")
	msg = strings.ReplaceAll(msg, logger.MagentaBold, "")
	msg = strings.ReplaceAll(msg, logger.RedBold, "")
	msg = strings.ReplaceAll(msg, logger.YellowBold, "")
	logOutFile(msg) // 输出到文件
}

func logOut(msg, level string) {
	start := time.Now()
	_, file, line, _ := runtime.Caller(2)
	file = filepath.Base(file)
	logMsg := fmt.Sprintf("[%s] - %s ==> [%s:%d] [%s] %s\n",
		time.Now().Format("2006-01-02 15:04:05"),
		time.Since(start),
		file,
		line,
		level,
		msg,
	)
	fmt.Print(logMsg)
	logOutFile(msg) // 输出到文件
}

func logOutFile(msg string) {
	logsDir := "logs/"
	e := os.MkdirAll(logsDir, 0644)
	if e != nil {
		return
	}
	filename := logsDir + time.Now().Format("2006-01-02") + ".log"
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("open log file error: ", err)
	}
	defer file.Close()
	//write
	if _, err := file.WriteString(msg); err != nil {
		log.Println("write log file error: ", err)
	}
}
