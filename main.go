package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-ini/ini"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type Config struct {
	GitRepoPath   string
	LocalCodePath string
	TimeLimitSec  int
}

var (
	endTime   time.Time
	mutex     sync.Mutex
	isRunning bool
)

func loadConfig() (*Config, error) {
	cfg, err := ini.Load("config.ini")
	if err != nil {
		return nil, err
	}

	config := &Config{
		GitRepoPath:   cfg.Section("Settings").Key("git_repo_path").String(),
		LocalCodePath: cfg.Section("Settings").Key("local_code_path").String(),
		TimeLimitSec:  cfg.Section("Settings").Key("time_limit_seconds").MustInt(),
	}

	return config, nil
}

func deleteFilesInDir(dirPath string) {
	fmt.Printf("Deleting files in directory %s\n", dirPath)
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fmt.Printf("Removing file: %s\n", path)
			os.Remove(path)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error cleaning directory: %v\n", err)
	} else {
		fmt.Println("Directory cleaned successfully.")
	}
}

func startCountdown(config *Config) {
	mutex.Lock()
	defer mutex.Unlock()

	if isRunning {
		return
	}

	isRunning = true
	endTime = time.Now().Add(time.Duration(config.TimeLimitSec) * time.Second)
	fmt.Printf("Countdown started. Code will be deleted at: %s\n", endTime.Format("2006-01-02 15:04:05"))

	go func() {
		for {
			time.Sleep(1 * time.Second)
			mutex.Lock()

			if time.Now().After(endTime) {
				// 删除指定目录中的文件
				deleteFilesInDir(config.LocalCodePath)
				deleteFilesInDir(config.GitRepoPath)
				isRunning = false
				mutex.Unlock()
				break
			}

			mutex.Unlock()
		}
	}()
}

func getRemainingTime() int {
	mutex.Lock()
	defer mutex.Unlock()

	if !isRunning {
		return 0
	}

	remaining := int(endTime.Sub(time.Now()).Seconds())
	if remaining < 0 {
		remaining = 0
	}

	return remaining
}

func main() {
	// 加载配置
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return
	}

	// 初始化 Gin 服务器
	router := gin.Default()

	// 提供静态文件（前端 HTML）
	router.StaticFile("/", "./index.html")

	// 提供 API 接口来获取剩余时间
	router.GET("/time", func(c *gin.Context) {
		remainingTime := getRemainingTime()
		if remainingTime == 0 {
			c.String(http.StatusOK, "0")
		} else {
			c.String(http.StatusOK, strconv.Itoa(remainingTime))
		}
	})

	// 提供 API 接口来启动删除倒计时
	router.POST("/delCode", func(c *gin.Context) {
		startCountdown(config)
		c.String(http.StatusOK, "Countdown started. Files will be deleted after the countdown ends.")
	})

	// 启动服务器
	router.Run(":8080")
}
