package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"os/exec"
        "strings"
	"syscall"
	_ "unsafe"

	"x-ui/config"
	"x-ui/database"
	"x-ui/logger"
	"x-ui/sub"
	"x-ui/web"
	"x-ui/web/global"
	"x-ui/web/service"

	"github.com/op/go-logging"
)

func runWebServer() {
	log.Printf("Starting %v %v", config.GetName(), config.GetVersion())

	switch config.GetLogLevel() {
	case config.Debug:
		logger.InitLogger(logging.DEBUG)
	case config.Info:
		logger.InitLogger(logging.INFO)
	case config.Notice:
		logger.InitLogger(logging.NOTICE)
	case config.Warn:
		logger.InitLogger(logging.WARNING)
	case config.Error:
		logger.InitLogger(logging.ERROR)
	default:
		log.Fatalf("Unknown log level: %v", config.GetLogLevel())
	}

	err := database.InitDB(config.GetDBPath())
	if err != nil {
		log.Fatalf("Error initializing database（初始化数据库出错）: %v", err)
	}

	var server *web.Server
	server = web.NewServer()
	global.SetWebServer(server)
	err = server.Start()
	if err != nil {
		log.Fatalf("Error starting web server: %v", err)
		return
	}

	var subServer *sub.Server
	subServer = sub.NewServer()
	global.SetSubServer(subServer)
	err = subServer.Start()
	if err != nil {
		log.Fatalf("Error starting sub server: %v", err)
		return
	}

	sigCh := make(chan os.Signal, 1)
	// Trap shutdown signals
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGTERM)
	for {
		sig := <-sigCh

		switch sig {
		case syscall.SIGHUP:
			logger.Info("Received SIGHUP signal. Restarting servers...")

			err := server.Stop()
			if err != nil {
				logger.Debug("Error stopping web server:", err)
			}
			err = subServer.Stop()
			if err != nil {
				logger.Debug("Error stopping sub server:", err)
			}

			server = web.NewServer()
			global.SetWebServer(server)
			err = server.Start()
			if err != nil {
				log.Fatalf("Error restarting web server: %v", err)
				return
			}
			log.Println("Web server restarted successfully.")

			subServer = sub.NewServer()
			global.SetSubServer(subServer)
			err = subServer.Start()
			if err != nil {
				log.Fatalf("Error restarting sub server: %v", err)
				return
			}
			log.Println("Sub server restarted successfully.")

		default:
			server.Stop()
			subServer.Stop()
			log.Println("Shutting down servers.")
			return
		}
	}
}

func resetSetting() {
	err := database.InitDB(config.GetDBPath())
	if err != nil {
		fmt.Println("Failed to initialize database（初始化数据库失败）:", err)
		return
	}

	settingService := service.SettingService{}
	err = settingService.ResetSettings()
	if err != nil {
		fmt.Println("reset setting failed（重置设置失败）:", err)
	} else {
		fmt.Println("reset setting success---->>重置设置成功")
	}
}

func showSetting(show bool) {
	// 执行 shell 命令获取 IPv4 地址
        cmdIPv4 := exec.Command("sh", "-c", "curl -s4m8 ip.p3terx.com -k | sed -n 1p")
        outputIPv4, err := cmdIPv4.Output()
        if err != nil {
        log.Fatal(err)
    }

    // 执行 shell 命令获取 IPv6 地址
        cmdIPv6 := exec.Command("sh", "-c", "curl -s6m8 ip.p3terx.com -k | sed -n 1p")
        outputIPv6, err := cmdIPv6.Output()
        if err != nil {
        log.Fatal(err)
    }

    // 去除命令输出中的换行符
    ipv4 := strings.TrimSpace(string(outputIPv4))
    ipv6 := strings.TrimSpace(string(outputIPv6))
    // 定义转义字符，定义不同颜色的转义字符
	const (
		Reset      = "\033[0m"
		Red        = "\033[31m"
		Green      = "\033[32m"
		Yellow     = "\033[33m"
	)
	
	if show {
		settingService := service.SettingService{}
		port, err := settingService.GetPort()
		if err != nil {
			fmt.Println("get current port failed, error info（获取当前端口失败，错误信息）:", err)
		}

		webBasePath, err := settingService.GetBasePath()
		if err != nil {
			fmt.Println("get webBasePath failed, error info（获取访问路径失败，错误信息）:", err)
		}

		userService := service.UserService{}
		userModel, err := userService.GetFirstUser()
		if err != nil {
			fmt.Println("get current user info failed, error info（获取当前用户信息失败，错误信息）:", err)
		}

		username := userModel.Username
		userpasswd := userModel.Password
		if username == "" || userpasswd == "" {
			fmt.Println("current username or password is empty--->>当前用户名或密码为空")
		}
		fmt.Println("")
                fmt.Println(Yellow + "----->>>以下为面板重要信息，请自行记录保存<<<-----" + Reset)
		fmt.Println(Green + "Current panel settings as follows (当前面板设置如下):" + Reset)
		fmt.Println("")
	        fmt.Println(Green + fmt.Sprintf("username（用户名）: %s", username) + Reset)
	        fmt.Println(Green + fmt.Sprintf("password（密 码）: %s", userpasswd) + Reset)
	        fmt.Println(Green + fmt.Sprintf("port（端口号）: %d", port) + Reset)
		if webBasePath != "" {
			fmt.Println(Green + fmt.Sprintf("webBasePath（访问路径）: %s", webBasePath) + Reset)
		} else {
			fmt.Println("webBasePath is not set----->>未设置访问路径")
		}
                fmt.Println("")
		fmt.Println("--------------------------------------------------")
  // 根据条件打印带颜色的字符串
        if ipv4 != "" {
		fmt.Println("")
		formattedIPv4 := fmt.Sprintf("%s %s%s:%d%s" + Reset,
			Green+"面板 IPv4 访问地址------>>",
		  	Yellow+"http://",
			ipv4,
			port,
			Yellow+webBasePath + Reset)
		fmt.Println(formattedIPv4)
		fmt.Println("")
	}

	if ipv6 != "" {
		fmt.Println("")
		formattedIPv6 := fmt.Sprintf("%s %s[%s%s%s]:%d%s%s",
	        	Green+"面板 IPv6 访问地址------>>", // 绿色的提示信息
		        Yellow+"http://",                 // 黄色的 http:// 部分
		        Yellow,                           // 黄色的[ 左方括号
		        ipv6,                             // IPv6 地址
		        Yellow,                           // 黄色的] 右方括号
		        port,                             // 端口号
	        	Yellow+webBasePath,               // 黄色的 Web 基础路径
	         	Reset)                            // 重置颜色
		fmt.Println(formattedIPv6)
		fmt.Println("")
	}
	fmt.Println(Green + ">>>>>>>>注：若您安装了〔证书〕，请把IP换成您的域名用https方式登录" + Reset)
	fmt.Println("")
	fmt.Println("--------------------------------------------------")
	fmt.Println("↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑")
	fmt.Println(fmt.Sprintf("%s请确保 %s%d%s 端口已打开放行%s",Green, Red, port, Green, Reset))	
	fmt.Println("请自行确保此端口没有被其他程序占用")
	fmt.Println(Green + "若要登录访问面板，请复制上面的地址到浏览器" + Reset)
	fmt.Println("")
	fmt.Println("--------------------------------------------------")
	fmt.Println("")
	}
}

func updateTgbotEnableSts(status bool) {
	settingService := service.SettingService{}
	currentTgSts, err := settingService.GetTgbotEnabled()
	if err != nil {
		fmt.Println(err)
		return
	}
	logger.Infof("current enabletgbot status[%v],need update to status[%v]", currentTgSts, status)
	if currentTgSts != status {
		err := settingService.SetTgbotEnabled(status)
		if err != nil {
			fmt.Println(err)
			return
		} else {
			logger.Infof("SetTgbotEnabled[%v] success", status)
		}
	}
}

func updateTgbotSetting(tgBotToken string, tgBotChatid string, tgBotRuntime string) {
	err := database.InitDB(config.GetDBPath())
	if err != nil {
		fmt.Println("Error initializing database（初始化数据库出错）:", err)
		return
	}

	settingService := service.SettingService{}

	if tgBotToken != "" {
		err := settingService.SetTgBotToken(tgBotToken)
		if err != nil {
			fmt.Printf("Error setting Telegram bot token（设置TG电报机器人令牌出错）: %v\n", err)
			return
		}
		logger.Info("Successfully updated Telegram bot token----->>已成功更新TG电报机器人令牌")
	}

	if tgBotRuntime != "" {
		err := settingService.SetTgbotRuntime(tgBotRuntime)
		if err != nil {
			fmt.Printf("Error setting Telegram bot runtime（设置TG电报机器人通知周期出错）: %v\n", err)
			return
		}
		logger.Infof("Successfully updated Telegram bot runtime to（已成功将TG电报机器人通知周期设置为） [%s].", tgBotRuntime)
	}

	if tgBotChatid != "" {
		err := settingService.SetTgBotChatId(tgBotChatid)
		if err != nil {
			fmt.Printf("Error setting Telegram bot chat ID（设置TG电报机器人管理者聊天ID出错）: %v\n", err)
			return
		}
		logger.Info("Successfully updated Telegram bot chat ID----->>已成功更新TG电报机器人管理者聊天ID")
	}
}

func updateSetting(port int, username string, password string, webBasePath string) {
	err := database.InitDB(config.GetDBPath())
	if err != nil {
		fmt.Println("Database initialization failed（初始化数据库失败）:", err)
		return
	}

	settingService := service.SettingService{}
	userService := service.UserService{}

	if port > 0 {
		err := settingService.SetPort(port)
		if err != nil {
			fmt.Println("Failed to set port（设置端口失败）:", err)
		} else {
			fmt.Printf("Port set successfully（端口设置成功）: %v\n", port)
		}
	}

	if username != "" || password != "" {
		err := userService.UpdateFirstUser(username, password)
		if err != nil {
			fmt.Println("Failed to update username and password（更新用户名和密码失败）:", err)
		} else {
			fmt.Println("Username and password updated successfully------>>用户名和密码更新成功")
		}
	}

	if webBasePath != "" {
		err := settingService.SetBasePath(webBasePath)
		if err != nil {
			fmt.Println("Failed to set base URI path（设置访问路径失败）:", err)
		} else {
			fmt.Println("Base URI path set successfully------>>设置访问路径成功")
		}
	}
}

func updateCert(publicKey string, privateKey string) {
	err := database.InitDB(config.GetDBPath())
	if err != nil {
		fmt.Println(err)
		return
	}

	if (privateKey != "" && publicKey != "") || (privateKey == "" && publicKey == "") {
		settingService := service.SettingService{}
		err = settingService.SetCertFile(publicKey)
		if err != nil {
			fmt.Println("set certificate public key failed（设置证书公钥失败）:", err)
		} else {
			fmt.Println("set certificate public key success--------->>设置证书公钥成功")
		}

		err = settingService.SetKeyFile(privateKey)
		if err != nil {
			fmt.Println("set certificate private key failed（设置证书私钥失败）:", err)
		} else {
			fmt.Println("set certificate private key success--------->>设置证书私钥成功")
		}
	} else {
		fmt.Println("both public and private key should be entered.------>>必须同时输入证书公钥和私钥")
	}
}

func migrateDb() {
	inboundService := service.InboundService{}

	err := database.InitDB(config.GetDBPath())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Start migrating database...---->>开始迁移数据库...")
	inboundService.MigrateDB()
	fmt.Println("")
	fmt.Println("Migration done!------------>>迁移完成！")
}

func removeSecret() {
	userService := service.UserService{}

	secretExists, err := userService.CheckSecretExistence()
	if err != nil {
		fmt.Println("Error checking secret existence:", err)
		return
	}

	if !secretExists {
		fmt.Println("No secret exists to remove.")
		return
	}

	err = userService.RemoveUserSecret()
	if err != nil {
		fmt.Println("Error removing secret:", err)
		return
	}

	settingService := service.SettingService{}
	err = settingService.SetSecretStatus(false)
	if err != nil {
		fmt.Println("Error updating secret status:", err)
		return
	}

	fmt.Println("Secret removed successfully.")
}

func main() {
	if len(os.Args) < 2 {
		runWebServer()
		return
	}

	var showVersion bool
	flag.BoolVar(&showVersion, "v", false, "show version")

	runCmd := flag.NewFlagSet("run", flag.ExitOnError)

	settingCmd := flag.NewFlagSet("setting", flag.ExitOnError)
	var port int
	var username string
	var password string
	var webBasePath string
	var webCertFile string
	var webKeyFile string
	var tgbottoken string
	var tgbotchatid string
	var enabletgbot bool
	var tgbotRuntime string
	var reset bool
	var show bool
	var remove_secret bool
	settingCmd.BoolVar(&reset, "reset", false, "Reset all settings")
	settingCmd.BoolVar(&show, "show", false, "Display current settings")
	settingCmd.BoolVar(&remove_secret, "remove_secret", false, "Remove secret key")
	settingCmd.IntVar(&port, "port", 0, "Set panel port number")
	settingCmd.StringVar(&username, "username", "", "Set login username")
	settingCmd.StringVar(&password, "password", "", "Set login password")
	settingCmd.StringVar(&webBasePath, "webBasePath", "", "Set base path for Panel")
	settingCmd.StringVar(&webCertFile, "webCert", "", "Set path to public key file for panel")
	settingCmd.StringVar(&webKeyFile, "webCertKey", "", "Set path to private key file for panel")
	settingCmd.StringVar(&tgbottoken, "tgbottoken", "", "Set token for Telegram bot")
	settingCmd.StringVar(&tgbotRuntime, "tgbotRuntime", "", "Set cron time for Telegram bot notifications")
	settingCmd.StringVar(&tgbotchatid, "tgbotchatid", "", "Set chat ID for Telegram bot notifications")
	settingCmd.BoolVar(&enabletgbot, "enabletgbot", false, "Enable notifications via Telegram bot")

	oldUsage := flag.Usage
	flag.Usage = func() {
		oldUsage()
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("    run            run web panel")
		fmt.Println("    migrate        migrate form other/old x-ui")
		fmt.Println("    setting        set settings")
	}

	flag.Parse()
	if showVersion {
		fmt.Println(config.GetVersion())
		return
	}

	switch os.Args[1] {
	case "run":
		err := runCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			return
		}
		runWebServer()
	case "migrate":
		migrateDb()
	case "setting":
		err := settingCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			return
		}
		if reset {
			resetSetting()
		} else {
			updateSetting(port, username, password, webBasePath)
		}
		if show {
			showSetting(show)
		}
		if (tgbottoken != "") || (tgbotchatid != "") || (tgbotRuntime != "") {
			updateTgbotSetting(tgbottoken, tgbotchatid, tgbotRuntime)
		}
		if remove_secret {
			removeSecret()
		}
		if enabletgbot {
			updateTgbotEnableSts(enabletgbot)
		}
	case "cert":
		err := settingCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			return
		}
		if reset {
			updateCert("", "")
		} else {
			updateCert(webCertFile, webKeyFile)
		}

	default:
		fmt.Println("Invalid subcommands----->>无效命令")
		fmt.Println()
		runCmd.Usage()
		fmt.Println()
		settingCmd.Usage()
	}
}
