package job

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"time"
	"sync"
                 "crypto/rand"
                 "encoding/hex"

	"x-ui/database"
	"x-ui/database/model"
	"x-ui/logger"
	"x-ui/xray"
                 "x-ui/web/service"
)

// =================================================================
// 中文注释: 以下是用于实现设备限制功能的核心代码
// =================================================================

// ActiveClientIPs 中文注释: 用于在内存中跟踪每个用户的活跃IP (TTL机制)
// 结构: map[用户email] -> map[IP地址] -> 最后活跃时间
var ActiveClientIPs = make(map[string]map[string]time.Time)
var activeClientsLock sync.RWMutex

// ClientStatus 中文注释: 用于跟踪每个用户的状态（是否因为设备超限而被禁用）
// 结构: map[用户email] -> 是否被禁用(true/false)
var ClientStatus = make(map[string]bool)
var clientStatusLock sync.RWMutex

// CheckDeviceLimitJob 中文注释: 这是我们的设备限制任务的结构体
type CheckDeviceLimitJob struct {
	inboundService service.InboundService
	xrayService    *service.XrayService
	// 中文注释: 新增 xrayApi 字段，用于持有 Xray API 客户端实例
	xrayApi xray.XrayAPI
	// lastPosition 中文注释: 用于记录上次读取 access.log 的位置，避免重复读取
	lastPosition int64
}

// RandomUUID 中文注释: 新增一个辅助函数，用于生成一个随机的 UUID
func RandomUUID() string {
	uuid := make([]byte, 16)
	rand.Read(uuid)
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return hex.EncodeToString(uuid[0:4]) + "-" + hex.EncodeToString(uuid[4:6]) + "-" + hex.EncodeToString(uuid[6:8]) + "-" + hex.EncodeToString(uuid[8:10]) + "-" + hex.EncodeToString(uuid[10:16])
}

// NewCheckDeviceLimitJob 中文注释: 创建一个新的任务实例
func NewCheckDeviceLimitJob(xrayService *service.XrayService) *CheckDeviceLimitJob {
	return &CheckDeviceLimitJob{
		xrayService: xrayService,
		// 中文注释: 初始化 xrayApi 字段
		xrayApi: xray.XrayAPI{},
	}
}

// Run 中文注释: 定时任务的主函数，每次定时器触发时执行
func (j *CheckDeviceLimitJob) Run() {
	// 中文注释: 检查 xray 是否正在运行，如果xray没运行，则无需执行此任务
	if !j.xrayService.IsXrayRunning() {
		return
	}

	// 1. 清理过期的IP
	j.cleanupExpiredIPs()

	// 2. 解析新的日志并更新IP列表
	j.parseAccessLog()

	// 3. 检查所有用户的设备限制状态
	j.checkAllClientsLimit()
}

// cleanupExpiredIPs 中文注释: 清理长时间不活跃的IP
func (j *CheckDeviceLimitJob) cleanupExpiredIPs() {
	activeClientsLock.Lock()
	defer activeClientsLock.Unlock()

	now := time.Now()
	// 中文注释: 活跃判断窗口(TTL): 近3分钟内出现过就算“活跃”
	const activeTTL = 3 * time.Minute
	for email, ips := range ActiveClientIPs {
		for ip, lastSeen := range ips {
			// 中文注释: 如果一个IP超过3分钟没有新的连接日志，我们就认为它已经下线
			if now.Sub(lastSeen) > activeTTL {
				delete(ActiveClientIPs[email], ip)
			}
		}
		// 中文注释: 如果一个用户的所有IP都下线了，就从大Map中移除这个用户，节省内存
		if len(ActiveClientIPs[email]) == 0 {
			delete(ActiveClientIPs, email)
		}
	}
}

// parseAccessLog 中文注释: 解析 xray access log 来获取最新的用户IP信息
func (j *CheckDeviceLimitJob) parseAccessLog() {
	logPath, err := xray.GetAccessLogPath()
	if err != nil || logPath == "none" || logPath == "" {
		return
	}

	file, err := os.Open(logPath)
	if err != nil {
		return
	}
	defer file.Close()

	// 中文注释: 移动到上次读取结束的位置，实现增量读取
	file.Seek(j.lastPosition, 0)

	scanner := bufio.NewScanner(file)

	// 中文注释: 使用正则表达式从日志行中提取 email 和 IP
	emailRegex := regexp.MustCompile(`email: ([^ ]+)`)
	ipRegex := regexp.MustCompile(`from (?:tcp:|udp:)?\[?([0-9a-fA-F\.:]+)\]?:\d+ accepted`)

	activeClientsLock.Lock()
	defer activeClientsLock.Unlock()

	now := time.Now()
	for scanner.Scan() {
		line := scanner.Text()
		
		emailMatch := emailRegex.FindStringSubmatch(line)
		ipMatch := ipRegex.FindStringSubmatch(line)

		if len(emailMatch) > 1 && len(ipMatch) > 1 {
			email := emailMatch[1]
			ip := ipMatch[1]

			if ip == "127.0.0.1" || ip == "::1" {
				continue
			}

			if _, ok := ActiveClientIPs[email]; !ok {
				ActiveClientIPs[email] = make(map[string]time.Time)
			}
			ActiveClientIPs[email][ip] = now
		}
	}

	currentPosition, err := file.Seek(0, os.SEEK_END)
	if err == nil {
		if currentPosition < j.lastPosition {
			j.lastPosition = 0
		} else {
			j.lastPosition = currentPosition
		}
	}
}

// checkAllClientsLimit 中文注释: 核心功能，检查所有用户，对超限的执行封禁，对恢复的执行解封
func (j *CheckDeviceLimitJob) checkAllClientsLimit() {
	db := database.GetDB()
	var inbounds []*model.Inbound
	// 中文注释: 这里仅查询启用了设备限制(device_limit > 0)并且自身是开启状态的入站规则
	db.Where("device_limit > 0 AND enable = ?", true).Find(&inbounds)

	if len(inbounds) == 0 {
		return
	}

	// 中文注释: 获取 API 端口。如果端口为0 (说明Xray未完全启动或有问题)，则直接返回
	apiPort := j.xrayService.GetApiPort()
	if apiPort == 0 {
		return
	}
	// 中文注释: 使用获取到的端口号初始化 API 客户端
	j.xrayApi.Init(apiPort)
	defer j.xrayApi.Close()

	// 中文注释: 优化 - 在一次循环中同时获取 tag 和 protocol
	inboundInfo := make(map[int]struct {
		Limit    int
		Tag      string
		Protocol model.Protocol
	})
	for _, inbound := range inbounds {
		inboundInfo[inbound.Id] = struct {
			Limit    int
			Tag      string
			Protocol model.Protocol
		}{Limit: inbound.DeviceLimit, Tag: inbound.Tag, Protocol: inbound.Protocol}
	}

	activeClientsLock.RLock()
	clientStatusLock.Lock()
	defer activeClientsLock.RUnlock()
	defer clientStatusLock.Unlock()

	for email, ips := range ActiveClientIPs {
		traffic, err := j.inboundService.GetClientTrafficByEmail(email)
		if err != nil || traffic == nil {
			continue
		}

		info, ok := inboundInfo[traffic.InboundId]
		if !ok || info.Limit <= 0 {
			continue
		}


		isBanned := ClientStatus[email]
		activeIPCount := len(ips)

		// 情况一: IP数量超限，且用户当前未被封禁 -> 执行封禁 (UUID 替换)
		if activeIPCount > info.Limit && !isBanned {
			_, client, err := j.inboundService.GetClientByEmail(email)
			if err != nil || client == nil {
				continue
			}
			logger.Infof("设备限制超限: 用户 %s. 限制: %d, 当前活跃: %d. 禁用该用户。", email, info.Limit, activeIPCount)

			// 中文注释: 步骤一：先从 Xray-Core 中删除该用户。
			err = j.xrayApi.RemoveUser(info.Tag, email)
			if err != nil {
				// 中文注释: 即便删除失败（例如，因为某些原因Xray内存中已无此用户），也应继续尝试添加封禁用的临时用户。
				logger.Warningf("尝试为用户 %s 启动设备限制时，删除原始用户失败(可忽略): %v", email, err)
			}

			// 中文注释: 创建一个带有随机UUID/Password的临时客户端配置用于“封禁”
			tempClient := *client
                                                   
                                                   // 适用于 VMess/VLESS
			if tempClient.ID != "" {
				tempClient.ID = RandomUUID()
			}

                                                   // 适用于 Trojan/Shadowsocks/Socks
			if tempClient.Password != "" {
				tempClient.Password = RandomUUID()
			}
			var clientMap map[string]interface{}
			clientJson, _ := json.Marshal(tempClient)
			json.Unmarshal(clientJson, &clientMap)

			// 中文注释: 步骤二：将这个带有错误UUID/Password的临时用户添加回去。
			// 客户端持有的还是旧的UUID，自然就无法通过验证，从而达到了“封禁”的效果。
			err = j.xrayApi.AddUser(string(info.Protocol), info.Tag, clientMap)
			if err != nil {
				logger.Warningf("通过API限制用户 %s 失败: %v", email, err)
			} else {
				// 中文注释: 封禁成功后，在内存中标记该用户为“已封禁”状态。
				ClientStatus[email] = true
			}
		}

		// 情况二: IP数量已恢复正常，但用户当前处于封禁状态 -> 执行解封 (恢复原始 UUID)
		if activeIPCount <= info.Limit && isBanned {
			_, client, err := j.inboundService.GetClientByEmail(email)
			if err != nil || client == nil {
				continue
			}
			logger.Infof("设备数量已恢复: 用户 %s. 限制: %d, 当前活跃: %d. 恢复用户。", email, info.Limit, activeIPCount)

			// 中文注释: 步骤一：先从 Xray-Core 中删除用于“封禁”的那个临时用户。
			err = j.xrayApi.RemoveUser(info.Tag, email)
			if err != nil {
				// 中文注释: 同样，这里的删除失败也可以容忍，最终目的是恢复用户。
				logger.Warningf("尝试为用户 %s 解除设备限制时，删除临时用户失败(可忽略): %v", email, err)
			}

			var clientMap map[string]interface{}
			clientJson, _ := json.Marshal(client)
			json.Unmarshal(clientJson, &clientMap)

			// 中文注释: 步骤二：将数据库中原始的、正确的用户信息重新添加回 Xray-Core，从而实现“解封”。
			err = j.xrayApi.AddUser(string(info.Protocol), info.Tag, clientMap)
			if err != nil {
				logger.Warningf("通过API恢复用户 %s 失败: %v", email, err)
			} else {
				// 中文注释: 解封成功后，从内存中移除该用户的“已封禁”状态标记。
				delete(ClientStatus, email)
			}
		}
	}
}

type CheckClientIpJob struct {
	lastClear     int64
	disAllowedIps []string
}

var job *CheckClientIpJob

func NewCheckClientIpJob() *CheckClientIpJob {
	job = new(CheckClientIpJob)
	return job
}

func (j *CheckClientIpJob) Run() {
	if j.lastClear == 0 {
		j.lastClear = time.Now().Unix()
	}

	shouldClearAccessLog := false
	iplimitActive := j.hasLimitIp()
	f2bInstalled := j.checkFail2BanInstalled()
	isAccessLogAvailable := j.checkAccessLogAvailable(iplimitActive)

	if iplimitActive {
		if f2bInstalled && isAccessLogAvailable {
			shouldClearAccessLog = j.processLogFile()
		} else {
			if !f2bInstalled {
				logger.Warning("[LimitIP] Fail2Ban is not installed, Please install Fail2Ban from the x-ui bash menu.")
			}
		}
	}

	if shouldClearAccessLog || (isAccessLogAvailable && time.Now().Unix()-j.lastClear > 3600) {
		j.clearAccessLog()
	}
}

func (j *CheckClientIpJob) clearAccessLog() {
	logAccessP, err := os.OpenFile(xray.GetAccessPersistentLogPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	j.checkError(err)
	defer logAccessP.Close()

	accessLogPath, err := xray.GetAccessLogPath()
	j.checkError(err)

	file, err := os.Open(accessLogPath)
	j.checkError(err)
	defer file.Close()

	_, err = io.Copy(logAccessP, file)
	j.checkError(err)

	err = os.Truncate(accessLogPath, 0)
	j.checkError(err)

	j.lastClear = time.Now().Unix()
}

func (j *CheckClientIpJob) hasLimitIp() bool {
	db := database.GetDB()
	var inbounds []*model.Inbound

	err := db.Model(model.Inbound{}).Find(&inbounds).Error
	if err != nil {
		return false
	}

	for _, inbound := range inbounds {
		if inbound.Settings == "" {
			continue
		}

		settings := map[string][]model.Client{}
		json.Unmarshal([]byte(inbound.Settings), &settings)
		clients := settings["clients"]

		for _, client := range clients {
			limitIp := client.LimitIP
			if limitIp > 0 {
				return true
			}
		}
	}

	return false
}

func (j *CheckClientIpJob) processLogFile() bool {

	ipRegex := regexp.MustCompile(`from (?:tcp:|udp:)?\[?([0-9a-fA-F\.:]+)\]?:\d+ accepted`)
	emailRegex := regexp.MustCompile(`email: (.+)$`)

	accessLogPath, _ := xray.GetAccessLogPath()
	file, _ := os.Open(accessLogPath)
	defer file.Close()

	inboundClientIps := make(map[string]map[string]struct{}, 100)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		ipMatches := ipRegex.FindStringSubmatch(line)
		if len(ipMatches) < 2 {
			continue
		}

		ip := ipMatches[1]

		if ip == "127.0.0.1" || ip == "::1" {
			continue
		}

		emailMatches := emailRegex.FindStringSubmatch(line)
		if len(emailMatches) < 2 {
			continue
		}
		email := emailMatches[1]

		if _, exists := inboundClientIps[email]; !exists {
			inboundClientIps[email] = make(map[string]struct{})
		}
		inboundClientIps[email][ip] = struct{}{}
	}

	shouldCleanLog := false
	for email, uniqueIps := range inboundClientIps {

		ips := make([]string, 0, len(uniqueIps))
		for ip := range uniqueIps {
			ips = append(ips, ip)
		}
		sort.Strings(ips)

		clientIpsRecord, err := j.getInboundClientIps(email)
		if err != nil {
			j.addInboundClientIps(email, ips)
			continue
		}

		shouldCleanLog = j.updateInboundClientIps(clientIpsRecord, email, ips) || shouldCleanLog
	}

	return shouldCleanLog
}

func (j *CheckClientIpJob) checkFail2BanInstalled() bool {
	cmd := "fail2ban-client"
	args := []string{"-h"}
	err := exec.Command(cmd, args...).Run()
	return err == nil
}

func (j *CheckClientIpJob) checkAccessLogAvailable(iplimitActive bool) bool {
	accessLogPath, err := xray.GetAccessLogPath()
	if err != nil {
		return false
	}

	if accessLogPath == "none" || accessLogPath == "" {
		if iplimitActive {
			logger.Warning("[LimitIP] Access log path is not set, Please configure the access log path in Xray configs.")
		}
		return false
	}

	return true
}

func (j *CheckClientIpJob) checkError(e error) {
	if e != nil {
		logger.Warning("client ip job err:", e)
	}
}

func (j *CheckClientIpJob) getInboundClientIps(clientEmail string) (*model.InboundClientIps, error) {
	db := database.GetDB()
	InboundClientIps := &model.InboundClientIps{}
	err := db.Model(model.InboundClientIps{}).Where("client_email = ?", clientEmail).First(InboundClientIps).Error
	if err != nil {
		return nil, err
	}
	return InboundClientIps, nil
}

func (j *CheckClientIpJob) addInboundClientIps(clientEmail string, ips []string) error {
	inboundClientIps := &model.InboundClientIps{}
	jsonIps, err := json.Marshal(ips)
	j.checkError(err)

	inboundClientIps.ClientEmail = clientEmail
	inboundClientIps.Ips = string(jsonIps)

	db := database.GetDB()
	tx := db.Begin()

	defer func() {
		if err == nil {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	err = tx.Save(inboundClientIps).Error
	if err != nil {
		return err
	}
	return nil
}

func (j *CheckClientIpJob) updateInboundClientIps(inboundClientIps *model.InboundClientIps, clientEmail string, ips []string) bool {
	jsonIps, err := json.Marshal(ips)
	if err != nil {
		logger.Error("failed to marshal IPs to JSON:", err)
		return false
	}

	inboundClientIps.ClientEmail = clientEmail
	inboundClientIps.Ips = string(jsonIps)

	inbound, err := j.getInboundByEmail(clientEmail)
	if err != nil {
		logger.Errorf("failed to fetch inbound settings for email %s: %s", clientEmail, err)
		return false
	}

	if inbound.Settings == "" {
		logger.Debug("wrong data:", inbound)
		return false
	}

	settings := map[string][]model.Client{}
	json.Unmarshal([]byte(inbound.Settings), &settings)
	clients := settings["clients"]
	shouldCleanLog := false
	j.disAllowedIps = []string{}

	logIpFile, err := os.OpenFile(xray.GetIPLimitLogPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		logger.Errorf("failed to open IP limit log file: %s", err)
		return false
	}
	defer logIpFile.Close()
	log.SetOutput(logIpFile)
	log.SetFlags(log.LstdFlags)

	for _, client := range clients {
		if client.Email == clientEmail {
			limitIp := client.LimitIP

			if limitIp > 0 && inbound.Enable {
				shouldCleanLog = true

				if limitIp < len(ips) {
					j.disAllowedIps = append(j.disAllowedIps, ips[limitIp:]...)
					for i := limitIp; i < len(ips); i++ {
						log.Printf("[LIMIT_IP] Email = %s || SRC = %s", clientEmail, ips[i])
					}
				}
			}
		}
	}

	sort.Strings(j.disAllowedIps)

	if len(j.disAllowedIps) > 0 {
		logger.Debug("disAllowedIps:", j.disAllowedIps)
	}

	db := database.GetDB()
	err = db.Save(inboundClientIps).Error
	if err != nil {
		logger.Error("failed to save inboundClientIps:", err)
		return false
	}

	return shouldCleanLog
}

func (j *CheckClientIpJob) getInboundByEmail(clientEmail string) (*model.Inbound, error) {
	db := database.GetDB()
	inbound := &model.Inbound{}

	err := db.Model(&model.Inbound{}).Where("settings LIKE ?", "%"+clientEmail+"%").First(inbound).Error
	if err != nil {
		return nil, err
	}

	return inbound, nil
}
