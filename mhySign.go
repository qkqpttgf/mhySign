package main

import (
	//"bufio"
	"context"
	//"crypto/hmac"
	"crypto/md5"
	//"crypto/sha256"
	//"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var programName string
var programVersion string
var programAuthor string

// 用于处理命令行是什么操作
var dbFilePath string
var cmd_Web bool
var cmd_Admin bool
var cmd_Sign bool
var cmd_Reset bool

var quit chan int
var slash string
var userhome string
var databaseFilename string
var mainCtx context.Context
var mainCancel context.CancelFunc
var settingFields []string
var userFields []string
var cookieFields []string
var logFields []string
var datebaseFieldMap map[string]string
var games []string
var urls []map[string]Links

type Links struct {
	getRoleUrl string
	checkSignUrl string
	signUrl string
	act_id string
	signgame string
}

func main() {
	programName = "mhySign"
	programVersion = "0.1.3.20250412_1144"
	programAuthor = "ysun"
	conlog(passlog("程序启动") + "\n")
	fmt.Println("  版本：" + programVersion)
	defer conlog(warnlog("程序结束") + "\n")
	mainCtx, mainCancel = context.WithCancel(context.Background())
	quit = make(chan int, 1)
	defer close(quit)
	defer waitExiting()

	userhome = userHomeDir() + slash + ".config" + slash + programName
	databaseFilename = "mhy.db"

	if !parseCommandLine() {
		// 命令行不是期望的格式
		conlog(alertlog("命令不对。") + "\n")
		useage()
		return
	}
	if !existSqlite() {
		conlog("sqlite3有问题!\n")
		return
	}
	if !checkDatabase() {
		// 数据库不对
		return
	}

		initUrls()
	if cmd_Reset {
		resetAdminPassword()
		return
	}
	if cmd_Web {
		startSrv(wizAport())
		return
	}
	if cmd_Admin {
		startTmpSrv(wizAport()+1)
		return
	}
	if cmd_Sign {
		//fmt.Println(urls[0]["崩坏3"].getRoleUrl)
		ids, _ := readConfig("user", "id", 0)
		//fmt.Println("_" + ids + "_")
		if ids != "" {
			for _, id := range strSplitLine(ids) {
				startSign(id)
				time.Sleep(time.Second * 2)
			}
		} else {
			conlog(warnlog("没有用户！") + "\n")
		}

		return
	}

	
	// 没有指定操作，显示用法
	useage()
	return
}

func parseCommandLine() bool {
	configFile := false
	softPath := ""
	conlog("输入的命令行:\n")
	for argc, argv := range os.Args {
		fmt.Printf("  %d: %v\n", argc, argv)
		if configFile {
			dbFilePath = argv
			configFile = false
			continue
		}
		if argc == 0 {
			softPath = argv
			pos := strings.LastIndex(softPath, slash)
			if pos > -1 {
				softPath = softPath[0:pos+1]
			} else {
				softPath = ""
			}
			continue
		}
		if argv == "web" {
			cmd_Web = true
			continue
		}
		if argv == "admin" {
			cmd_Admin = true
			continue
		}
		if argv == "sign" {
			cmd_Sign = true
			continue
		}
		if argv == "reset" {
			cmd_Reset = true
			continue
		}
		if argv == "-config" || argv == "-c" {
			configFile = true
			continue
		}

		// not 
		conlog("未知参数: " + argv + "\n")
		return false
	}
	todoNum := 0
	if cmd_Web {
		todoNum++
	}
	if cmd_Admin {
		todoNum++
	}
	if cmd_Sign {
		todoNum++
	}
	if cmd_Reset {
		todoNum++
	}
	if todoNum > 1 {
		conlog(alertlog("一次请只做一件事\n"))
		return false
	}
	if dbFilePath == "" {
		os.MkdirAll(userhome, os.ModePerm)
		dbFilePath = userhome  + slash + databaseFilename
	}
	conlog("将使用数据库:\n  " + warnlog(dbFilePath) + "\n")
	return true
}
func useage() {
	html := `用法:
` + os.Args[0] + ` [-c path/datafile] [admin | web | sign | reset]
  -c|-config PathOfDBfile   指定database位置
  admin                     开启一个有时效的临时网站用来管理
  web                       开启一个网站，让用户注册填写等
  sign                      开始签到
  reset                     重置管理密码
`
	//fmt.Print(html)
	conlog(html)
}
func waitExiting() {
	select {
		case <- mainCtx.Done() :
			// ctx被杀掉了，说明有其它地方按ctrl c了
			return
		default :
			mainCancel() // 没人杀，我来杀
			expireSecond := 10
			go displayCountdown("等 ", expireSecond, "s 或按 ctrl+c 退出。", quit)
			go func() {
				waitSYS()
				quit <- -1
			}()
			<- quit
	}
}
func initUrls() {
	games = []string {"崩坏3", "原神", "星穹铁道", "绝区零"}
	urls_cn := make(map[string]Links)
	var bh3_cn Links
	bh3_cn.getRoleUrl = "https://api-takumi.mihoyo.com/binding/api/getUserGameRolesByCookie?game_biz=bh3_cn"
	bh3_cn.checkSignUrl="https://api-takumi.mihoyo.com/event/luna/info"
	bh3_cn.signUrl="https://api-takumi.mihoyo.com/event/luna/sign"
	bh3_cn.act_id="e202306201626331"
	bh3_cn.signgame="bh3"
	urls_cn["崩坏3"] = bh3_cn

	var hk4e_cn Links
	hk4e_cn.getRoleUrl="https://api-takumi.mihoyo.com/binding/api/getUserGameRolesByCookie?game_biz=hk4e_cn"
	hk4e_cn.checkSignUrl="https://api-takumi.mihoyo.com/event/luna/info"
	hk4e_cn.signUrl="https://api-takumi.mihoyo.com/event/luna/sign"
	hk4e_cn.act_id="e202311201442471"
	hk4e_cn.signgame="hk4e"
	urls_cn["原神"] = hk4e_cn

	var hkrpg_cn Links
	hkrpg_cn.getRoleUrl="https://api-takumi.mihoyo.com/binding/api/getUserGameRolesByCookie?game_biz=hkrpg_cn"
	hkrpg_cn.checkSignUrl="https://api-takumi.mihoyo.com/event/luna/info"
	hkrpg_cn.signUrl="https://api-takumi.mihoyo.com/event/luna/sign"
	hkrpg_cn.act_id="e202304121516551"
	hkrpg_cn.signgame="hkrpg"
	urls_cn["星穹铁道"] = hkrpg_cn

	var nap_cn Links
	nap_cn.getRoleUrl="https://api-takumi.mihoyo.com/binding/api/getUserGameRolesByCookie?game_biz=nap_cn"
	nap_cn.checkSignUrl="https://act-nap-api.mihoyo.com/event/luna/zzz/info"
	nap_cn.signUrl="https://act-nap-api.mihoyo.com/event/luna/zzz/sign"
	nap_cn.act_id="e202406242138391"
	nap_cn.signgame="zzz"
	urls_cn["绝区零"] = nap_cn

	urls_global := make(map[string]Links)
	var bh3_global Links
	bh3_global.getRoleUrl="https://api-os-takumi.mihoyo.com/binding/api/getUserGameRolesByLtoken?game_biz=bh3_global"
	bh3_global.checkSignUrl="https://sg-hk4e-api.hoyolab.com/event/luna/info"
	bh3_global.signUrl="https://sg-hk4e-api.hoyolab.com/event/luna/sign"
	bh3_global.act_id="e202110291205111"
	bh3_global.signgame="bh3"
	urls_global["崩坏3"] = bh3_global

	var hk4e_global Links
	hk4e_global.getRoleUrl="https://api-os-takumi.mihoyo.com/binding/api/getUserGameRolesByLtoken?game_biz=hk4e_global"
	hk4e_global.checkSignUrl="https://sg-hk4e-api.hoyolab.com/event/sol/info"
	hk4e_global.signUrl="https://sg-hk4e-api.hoyolab.com/event/sol/sign"
	hk4e_global.act_id="e202102251931481"
	hk4e_global.signgame="hk4e"
	urls_global["原神"] = hk4e_global

	var hkrpg_global Links
	hkrpg_global.getRoleUrl="https://sg-public-api.hoyolab.com/binding/api/getUserGameRolesByLtoken?game_biz=hkrpg_global"
	hkrpg_global.checkSignUrl="https://sg-public-api.hoyolab.com/event/luna/info"
	hkrpg_global.signUrl="https://sg-public-api.hoyolab.com/event/luna/sign"
	hkrpg_global.act_id="e202303301540311"
	hkrpg_global.signgame="hkrpg"
	urls_global["星穹铁道"] = hkrpg_global

	var nap_global Links
	nap_global.getRoleUrl="https://sg-public-api.hoyolab.com/binding/api/getUserGameRolesByLtoken?game_biz=nap_global"
	nap_global.checkSignUrl="https://sg-public-api.hoyolab.com/event/luna/zzz/info"
	nap_global.signUrl="https://sg-public-api.hoyolab.com/event/luna/zzz/sign"
	nap_global.act_id="e202406031448091"
	nap_global.signgame="zzz"
	urls_global["绝区零"] = nap_global

	urls = append(urls, urls_cn)
	urls = append(urls, urls_global)
}
func userHomeDir() string {
	if runtime.GOOS == "windows" {
		slash = "\\"
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	} else {
		slash = "/"
		return os.Getenv("HOME")
	}
	
}
func checkDatabase() bool {
	settingFields = []string {"label", "setting"}
	userFields = []string {"username", "password", "needResetPassword", "accountDisable", "enableSign", "workWeiBotKey", "dingDingBotToken", "SCTKey", "SC3Key", "cookieIDs"}
	cookieFields = []string {"label", "region", "cookie", "userID"}
	logFields = []string {"userID", "signTime", "log"}
	datebaseFieldMap = make(map[string]string)
	datebaseFieldMap["setting"] = "text"
	datebaseFieldMap["username"] = "text unique"
	datebaseFieldMap["password"] = "text"
	datebaseFieldMap["needResetPassword"] = "int"
	datebaseFieldMap["accountDisable"] = "int"
	datebaseFieldMap["enableSign"] = "int"
	datebaseFieldMap["workWeiBotKey"] = "text"
	datebaseFieldMap["dingDingBotToken"] = "text"
	datebaseFieldMap["SCTKey"] = "text"
	datebaseFieldMap["SC3Key"] = "text"
	datebaseFieldMap["cookieIDs"] = "text"
	datebaseFieldMap["label"] = "text"
	datebaseFieldMap["region"] = "int"
	datebaseFieldMap["cookie"] = "text"
	datebaseFieldMap["userID"] = "int"
	datebaseFieldMap["signTime"] = "text"
	datebaseFieldMap["log"] = "text"

	createTableSQL := "CREATE TABLE setting (id integer primary key"
	for _, key := range settingFields {
		createTableSQL += ", " + key + " " + datebaseFieldMap[key]
	}
	createTableSQL += ");\nCREATE TABLE user (id integer primary key"
	for _, key := range userFields {
		createTableSQL += ", " + key + " " + datebaseFieldMap[key]
	}
	createTableSQL += ");\nCREATE TABLE cookie (id integer primary key"
	for _, key := range cookieFields {
		createTableSQL += ", " + key + " " + datebaseFieldMap[key]
	}
	createTableSQL += ");\nCREATE TABLE log (id integer primary key"
	for _, key := range logFields {
		createTableSQL += ", " + key + " " + datebaseFieldMap[key]
	}
	createTableSQL += ");"
	_, err := os.Stat(dbFilePath)
	if err != nil {
		conlog("数据库文件" + alertlog("不存在") + "， 即将" + warnlog("创建") + "。\n")
		sqlArr := strSplitLine(createTableSQL)
		for _, sql := range sqlArr {
			fmt.Println("  " + sql)
			_, err = sqlite(sql)
			if err != nil {
				conlog(alertlog("创建失败...\n") + sql + "\n")
				return false
			}
		}
		conlog(passlog("数据库创建成功！\n"))
		return true
	} else {
		conlog("检查数据库...\n")
		errStr := alertlog("出错\n") + warnlog("  这可能不是 " + programName + " 的数据库，或者是老版本。\n  请使用其它文件，或将它删除或重命名以便程序重新创建新的。\n")
		create_Arr := strSplitLine(createTableSQL)
		existSql, _ := sqlite(".schema")
		exist_Arr := strSplitLine(existSql)
		if len(create_Arr) != len(exist_Arr) {
			conlog(errStr)
			return false
		}
		exist_Map := make(map[string]string)
		for _, sql := range exist_Arr {
			if sql != "" {
				exist_Map[sql] = "1"
			}
		}
		for _, sql := range create_Arr {
			if sql != "" {
				delete(exist_Map, sql)
			}
		}
		if len(exist_Map) == 0 {
			conlog(passlog("数据库正常\n"))
			return true
		} else {
			conlog(errStr)
			//fmt.Println("_" + createTableSQL + "_")
			//fmt.Println("_" + existSql + "_")
			return false
		}
	}
}
func resetAdminPassword() {
	passid := findConfig("setting", "label", "adminpass")[0]
	if passid < 0 {
		fmt.Println("没有设置过管理员，请正常创建管理员。")
		return
	}
	newPass := randomPassword()
	err := setSetting("adminpass", newPass)
	if err != nil {
		fmt.Println(alertlog(" 重置失败："), err)
		return
	}
	username := readSetting("adminuser")
	fmt.Println(passlog("  重置成功！"))
	fmt.Println("  用户名：", username)
	fmt.Println("  新密码：", newPass)
	return
}
func startSign(userid string) {
	NotifyMsg := ""
	layout := "2006-01-02 15:04:05 Monday"
	//startTime := (time.Now()).Format(layout)
	conlog("用户id " + userid + " 开始：\n")
	defer conlog("用户id " + userid + " 结束。\n")

	sql := "select "
	for _, key := range userFields {
		sql += key + ","
	}
	sql = sql[0:len(sql)-1]
	sql += " from user where id=" + userid + ";"
	//fmt.Println(sql)
	res1, err := sqlite(sql)
	if err != nil {
		fmt.Println("Something wrong.")
		fmt.Println(err)
		return
	}
	res := strings.Split(res1, "|")
	user := make(map[string]string)
	for i:=0;i<len(userFields);i++ {
		user[userFields[i]] = res[i]
	}
	if user["accountDisable"] == "1" {
		fmt.Println("  用户 " + user["username"] + " 账号被禁！")
		return
	}
	if user["enableSign"] != "1" {
		fmt.Println("  用户 " + user["username"] + " 禁止签到！")
		return
	}
	if user["cookieIDs"] == "" {
		fmt.Println("  用户 " + user["username"] + " 没有添加cookie！")
		return
	}

	//for _, id := range strings.FieldsFunc(user["cookieIDs"], func(r rune) bool { return r == ',' }) {
	for _, id := range strings.Split(user["cookieIDs"], ",") {
		conlog(" Cookie " + id + " 开始：\n")
		sql := "select "
		for _, key := range cookieFields {
			sql += key + ","
		}
		sql = sql[0:len(sql)-1]
		sql += " from cookie where id=" + id + ";"
		//fmt.Println(sql)
		res, err := sqlite(sql)
		if err != nil {
			fmt.Println("Something wrong.")
			fmt.Println(err)
			return
		}
		cookie := make(map[string]string)
		for _, key := range cookieFields {
			if strings.Index(res, "|") > -1 {
				cookie[key] = res[0:strings.Index(res, "|")]
				res = res[strings.Index(res, "|")+1:]
			} else {
				cookie[key] = res
			}
		}
		fmt.Println(" " + cookie["label"] + ",")
		NotifyMsg += cookie["label"] + ":\n"
		serverRegion, _ := strconv.Atoi(cookie["region"])
		for _, key := range games {
			fmt.Print("  " + key + "，")
			NotifyMsg += "  " + key + "，"
			head := make(map[string]string)
			head["Cookie"] = cookie["cookie"]
			res, err := curl("GET", urls[serverRegion][key].getRoleUrl, "",  head)
			time.Sleep(time.Second * 1)
			if err == nil && res.StatusCode == 200 {
				retcode := readValueInString(res.Body, "retcode")
				if retcode == "0" {
					numOfAccount := strings.Count(res.Body, "game_uid")
					fmt.Println("有" + fmt.Sprint(numOfAccount) + "个角色")
					NotifyMsg += "有" + fmt.Sprint(numOfAccount) + "个角色\n"
					tmp := res.Body[strings.Index(res.Body, "\"list\"")+4:]
					tmp = tmp[strings.Index(tmp, "[")+1:]
					for i:=0; i<numOfAccount; i++ {
						j:=i+1
						fmt.Print("   [" + fmt.Sprint(j) + "] ")
						NotifyMsg += "   [" + fmt.Sprint(j) + "] "
						account := tmp[0:strings.Index(tmp, "}")]
						tmp = tmp[strings.Index(tmp, "}")+1:]
						region := readValueInString(account, "region")
						region_name := readValueInString(account, "region_name")
						level := readValueInString(account, "level")
						nickname := readValueInString(account, "nickname")
						game_uid := readValueInString(account, "game_uid")
						fmt.Print(region_name + " " + level + "级的 " + nickname + "(" + game_uid + "): ")
						NotifyMsg += region_name + " " + level + "级的 " + nickname + "(" + game_uid + "): "
						checkedDay := signCheck(serverRegion, key, region, game_uid, cookie["cookie"])
						if checkedDay == "0" {
							signRes := sign(serverRegion, key, region, game_uid, cookie["cookie"])
							if signRes == "0" {
								checkedDay = signCheck(serverRegion, key, region, game_uid, cookie["cookie"])
								fmt.Println("签到成功，本月共签" + checkedDay + "天")
								NotifyMsg += "签到成功，本月共签" + checkedDay + "天\n"
							} else {
								fmt.Println(signRes)
								NotifyMsg += signRes + "\n"
							}
						} else {
							_, err = strconv.Atoi(checkedDay)
							if err != nil {
								fmt.Println(checkedDay)
								NotifyMsg += checkedDay + "\n"
							} else {
								fmt.Println("今日签过，本月已签" + checkedDay + "天")
								NotifyMsg += "今日签过，本月已签" + checkedDay + "天\n"
							}
						}
					}
				} else {
					fmt.Println("出错")
					message := readValueInString(res.Body, "message")
					fmt.Println(message)
					NotifyMsg += message + "\n"
				}
			} else {
				fmt.Println("网络问题")
				NotifyMsg += "网络问题\n"
			}
		}
		conlog(" Cookie " + id + " 结束。\n")
	}
	if NotifyMsg[len(NotifyMsg)-1:] == "\n" {
		NotifyMsg = NotifyMsg[0:len(NotifyMsg)-1]
	}
	logMsg := NotifyMsg
	NotifyTitle := "米游社签到"
	endTime := (time.Now()).Format(layout)
	NotifyMsg = endTime + "\n" + NotifyMsg

	if user["workWeiBotKey"] != "" {
		NotifyResult := WorkWeiBot(user["workWeiBotKey"], NotifyTitle + "\n" + NotifyMsg)
		fmt.Println("企业微信通知：", NotifyResult)
		logMsg += "\n" + "企业微信通知：" + NotifyResult
	} else {
		fmt.Println("未设置企业微信通知")
	}
	if user["dingDingBotToken"] != "" {
		NotifyResult := DingDingBot(user["dingDingBotToken"], NotifyTitle + "\n" + NotifyMsg)
		fmt.Println("钉钉通知：", NotifyResult)
		logMsg += "\n" + "钉钉通知：" + NotifyResult
	} else {
		fmt.Println("未设置钉钉通知")
	}
	if user["SCTKey"] != "" {
		NotifyResult := FTSC(user["SCTKey"], NotifyTitle, NotifyMsg)
		fmt.Println("Server酱T通知：", NotifyResult)
		logMsg += "\n" + "Server酱T通知：" + NotifyResult
	} else {
		fmt.Println("未设置Server酱T通知")
	}
	if user["SC3Key"] != "" {
		NotifyResult := FTSC3(user["SC3Key"], NotifyTitle, NotifyMsg)
		fmt.Println("Server酱3通知：", NotifyResult)
		logMsg += "\n" + "Server酱3通知：" + NotifyResult
	} else {
		fmt.Println("未设置Server酱3通知")
	}
	saveLog(userid, (time.Now()).Format(layout), logMsg)
}
func sign(serverRegion int, game string, region string, game_uid string, cookie string) string {
	data1 := `
{
	"act_id": "` + urls[serverRegion][game].act_id + `",
	"region": "` + region + `",
	"uid": "` + game_uid + `"
}`

	salt1 := "rtvTthKxEyreVXQCnhluFgLXPOFKPHlA"
	time1 := fmt.Sprint(time.Now().Unix())
	random1 := "ysun65"
	md51 := fmt.Sprintf("%x", md5.Sum([]byte("salt=" + salt1 + "&t=" + time1 + "&r=" + random1)))

	head := make(map[string]string)
	head["User-Agent"] = "Android; miHoYoBBS/2.71.1"
	head["Cookie"] = cookie
	head["Content-Type"] = "application/json"
	head["x-rpc-device_id"] = "F84E53D45BFE4424ABEA9D6F0205FF4A"
	head["x-rpc-app_version"] = "2.71.1"
	head["x-rpc-client_type"] = "5"
	head["DS"] = time1 + "," + random1 + "," + md51
	head["x-rpc-signgame"] = urls[serverRegion][game].signgame

	res, err := curl("POST", urls[serverRegion][game].signUrl, data1, head)
	time.Sleep(time.Second * 1)
	if err != nil {
		fmt.Print(err)
		return fmt.Sprint(err)
	} else {
		if res.StatusCode == 200 {
			retcode := readValueInString(res.Body, "retcode")
			if retcode == "0" {
				success := readValueInString(res.Body, "success")
				if success == "0" {
					//签到成功
					return "0"
				} else {
					is_risk := readValueInString(res.Body, "is_risk")
					if is_risk == "true" {
						return "有验证码"
					} else {
						return res.Body
					}
				}
			} else {
				if retcode == "-5003" {
					fmt.Print("已经签过")
				} else {
					fmt.Print("出错")
				}
				message := readValueInString(res.Body, "message")
				//fmt.Println(message)
				return message
			}
		} else {
			fmt.Println("网络问题")
			return "网络问题"
		}
	}
}
func signCheck(serverRegion int, game string, region string, game_uid string, cookie string) string {
	url := urls[serverRegion][game].checkSignUrl + "?act_id=" + urls[serverRegion][game].act_id + "&region=" + region + "&uid=" + game_uid
	head := make(map[string]string)
	head["Cookie"] = cookie
	head["x-rpc-signgame"] = urls[serverRegion][game].signgame
	res, err := curl("GET", url, "",  head)
	time.Sleep(time.Second * 1)
	if err != nil {
		fmt.Println(err)
		return fmt.Sprint(err)
	} else {
		if res.StatusCode == 200 {
			retcode := readValueInString(res.Body, "retcode")
			if retcode == "0" {
				is_sign := readValueInString(res.Body, "is_sign")
				if is_sign == "true" {
					total_sign_day := readValueInString(res.Body, "total_sign_day")
					return total_sign_day
				} else {
					return "0"
				}
			} else {
				fmt.Println("出错")
				message := readValueInString(res.Body, "message")
				fmt.Println(message)
				return message
			}
		} else {
			fmt.Println("网络问题")
			return "网络问题"
		}
	}
}
func checkMHYCookie(region string, cookie string) bool {
	/*url := "https://bbs-api.miyoushe.com/user/wapi/getUserFullInfo?gids=2"
	if region == "1" {
		url = "https://bbs-api-os.hoyolab.com/community/user/wapi/getUserFullInfo?gid=2" // 弃用，需要程序在外网
	}

	salt1 := "rtvTthKxEyreVXQCnhluFgLXPOFKPHlA"
	time1 := fmt.Sprint(time.Now().Unix())
	random1 := "ysun65"
	md51 := md5Sum("salt=" + salt1 + "&t=" + time1 + "&r=" + random1)

	head := make(map[string]string)
	head["User-Agent"] = "Android; miHoYoBBS/2.71.1"
	head["Cookie"] = cookie
	head["Content-Type"] = "application/json"
	head["x-rpc-device_id"] = "F84E53D45BFE4424ABEA9D6F0205FF4A"
	head["x-rpc-app_version"] = "2.71.1"
	head["x-rpc-client_type"] = "5"
	if region == "1" {
		head["referer"] = "https://act.hoyolab.com/"
	} else {
		head["referer"] = "https://www.miyoushe.com/"
	}
	head["DS"] = time1 + "," + random1 + "," + md51

	res, err := curl("GET", url, "", head)*/

	head := make(map[string]string)
	head["Cookie"] = cookie
	serverRegion, _ := strconv.Atoi(region)
	res, err := curl("GET", urls[serverRegion]["崩坏3"].getRoleUrl, "",  head)
	time.Sleep(time.Second * 1)
	if err == nil {
		//fmt.Println(res.Body)
		retcode := readValueInString(res.Body, "retcode")
		if retcode == "0" {
			return true
		} else {
			fmt.Println(res.Body)
		}
	} else {
		fmt.Println(err)
	}
	return false
}
func saveLog(userID string, signTime string, log string) {
	log = strings.ReplaceAll(log, "\n", "\\n")
	sql := `insert into log (userID, signTime, log) values ("` + userID + `", "` + signTime + `", '` + log + `');`
	res, err := sqlite(sql)
	if err != nil {
		fmt.Println(sql, "\n", res, err)
	}
}
func WorkWeiBot(key string, msg string) string {
	msg1 := strings.ReplaceAll(msg, "\"", "\\\"")
	url := "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=" + key
	head := make(map[string]string)
	head["Content-Type"] = "application/json"
	data1 := `
	{
		"msgtype": "text",
		"text": {
			"content": "` + msg1 + `"
		}
	}`
	res, _ := curl("POST", url, data1, head)
	//fmt.Println(res.Body)
	errcode := readValueInString(res.Body, "errcode")
	if errcode == "0" {
		return "成功"
	} else {
		return res.Body
	}
}
func DingDingBot(token string, msg string) string {
	msg1 := strings.ReplaceAll(msg, "\"", "")
	url := "https://oapi.dingtalk.com/robot/send?access_token=" + token
	head := make(map[string]string)
	head["Content-Type"] = "application/json"
	data1 := `
	{
		"msgtype": "text",
		"text": {
			"content": "` + msg1 + `"
		}
	}`
	res, _ := curl("POST", url, data1, head)
	errcode := readValueInString(res.Body, "errcode")
	if errcode == "0" {
		return "成功"
	} else {
		return res.Body
	}
}
func FTSC(key string, title string, msg string) string {
	msg1 := strings.ReplaceAll(msg, "\n", "\\n\\n")
	url := "https://sctapi.ftqq.com/" + key + ".send"
	head := make(map[string]string)
	head["Content-Type"] = "application/json"
	data1 := `{
	"text": "` + title + `",
	"desp": "` + msg1 + `"
}`
	res, _ := curl("POST", url, data1, head)
	//fmt.Println(strconv.Unquote(res.Body))
	errcode := readValueInString(res.Body, "code")
	if errcode == "0" {
		return "成功"
	} else {
		return res.Body
	}
}
func FTSC3(key string, title string, msg string) string {
	msg1 := strings.ReplaceAll(msg, "\n", "\\n\\n")
	uid := key[4:]
	uid = uid[0:strings.Index(uid, "t")]
	url := "https://" + uid + ".push.ft07.com/send/" + key + ".send"
	head := make(map[string]string)
	head["Content-Type"] = "application/json"
	data1 := `{
	"title": "` + title + `",
	"desp": "` + msg1 + `"
}`
	res, _ := curl("POST", url, data1, head)
	errcode := readValueInString(res.Body, "code")
	if errcode == "0" {
		return "成功"
	} else {
		return res.Body
	}
}
func getChar() string {
	b := make([]byte, 1)
	os.Stdin.Read(b)
	return string(b)
}
func trash_getChar() string {
	ch := make(chan string)
	defer close(ch)
	go func(ch chan string) {
		// disable input buffering
		exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run()
		//defer exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "10").Run()
		// do not display entered characters on the screen
		//exec.Command("stty", "-F", "/dev/tty", "-echo").Run()
		//defer exec.Command("stty", "-F", "/dev/tty", "echo").Run()
		var b []byte = make([]byte, 1)
		//for {
			os.Stdin.Read(b)
			ch <- string(b)
		//}
	}(ch)
	for {
		select {
			case stdin, _ := <-ch:
				return stdin
			default:
				// do nothing
		}
		time.Sleep(time.Millisecond * 100)
	}

}
func waitSYS() {
	// 等候系统中断，比如按ctrl c
	sysSignalQuit := make(chan os.Signal, 1)
	defer close(sysSignalQuit)
	signal.Notify(sysSignalQuit, syscall.SIGINT, syscall.SIGTERM)
	<- sysSignalQuit
	mainCancel()
	fmt.Print("\n")
}
func wait_Web() {
	go func() {
		waitSYS()
		quit <- -1
	}()
	// 底部跑马灯文字
	go displayHorseRaceLamp()
	// 等ctrl c
	<- quit
}
// 等管理操作，超时后程序直接结束，避免下次运行还在占用端口
// 每次访问后刷新超时时间
func wait_Admin(expireSecond int) {
	defer conlog("管理网页停止\n")
	count := 0
	// 底部跑马灯文字
	go displayHorseRaceLamp()
	go func() {
		waitSYS()
		quit <- -1
	}()
	// 判断超时
	for count > -1 {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(time.Second * time.Duration(expireSecond))
			// 超时时间后判断当前ctx是否存活
			select {
				case <- ctx.Done() :
					// ctx被杀掉了，说明有其它地方给quit传值了
					// 本routine结束不做操作
					return
				default :
					// ctx存活，说明其它地方无操作
					// 超时，给通道传值结束程序
					conlog("时间到\n")
					quit <- -1
			}
		}()
		// 等待网页路由触发传值，或上方Sleep后传值
		count = <- quit
		// 有值传入后，结束本次ctx，开始下一次循环
		cancel()
	}

}
// 跑马灯显示字符，死循环，未考虑结束
func displayHorseRaceLamp() {
	str := "Waiting visitor ..."
	runstr := []rune(str) // 以防有中文字
	//fmt.Println(len(str), len(runstr))
	count := 0
	for count > -1 {
		select {
			case <- mainCtx.Done() :
				return
			default :	
				fmt.Print("\r" + string(runstr[0:count]))
				time.Sleep(1 * time.Second)
				count++
				//fmt.Println(count)
				if count > len(runstr) {
					count = 0
					clearCurrentLine(str)
				}
		}
	}
}
// 倒计时
func displayCountdown(pre string, expireSecond int, aft string, quit chan int) {
	for expireSecond > -1 {
		str := pre + fmt.Sprint(expireSecond) + aft
		fmt.Print("\r" + str)
		time.Sleep(1 * time.Second)
		clearCurrentLine(str)
		expireSecond--
	}
	quit <- -1
}
func clearCurrentLine(str string) {
	width := screenWidth()
	if width == 0 {
		width = len(str)
	}
	fmt.Print("\r")
	for i := 0; i < width; i++ {
		fmt.Print(" ")
	}
}
func screenWidth() int {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("mode", "con")
		result_b, err := cmd.Output()
		if err == nil {
			result_a := strings.Split(string(result_b), ":")
			result := result_a[3]
			result = strSplitLine(result)[0]
			result = result[strings.LastIndex(result, " ")+1:]
			width, err := strconv.Atoi(result)
			if err == nil {
				//if isCmdWindow {
				//	width--
				//}
				return width
			}
		}
	} else {
		cmd := exec.Command("tput", "cols")
		result_b, err := cmd.Output()
		if err == nil {
			result := strings.TrimSpace(string(result_b))
			width, err := strconv.Atoi(result)
			if err == nil {
				return width
			}
		}
	}
	return 0
}

func strSplitLine(target string) []string {
	return strings.FieldsFunc(target, func(r rune) bool {
        return r == '\r' || r == '\n'
    })
}

func conlog(log string) {
	layout := "[2006-01-02 15:04:05.000] "
	strTime := (time.Now()).Format(layout)
	fmt.Print("\r", strTime, log)
}
func alertlog(log string) string {
	return fmt.Sprintf("\033[91;5m%s\033[0m", log)
}
func warnlog(log string) string {
	//return fmt.Sprintf("\033[92;93m%s\033[0m", log)
	return fmt.Sprintf("\033[33;33m%s\033[0m", log)
}
func passlog(log string) string {
	return fmt.Sprintf("\033[92;32m%s\033[0m", log)
	//return fmt.Sprintf("\033[92;60m%s\033[0m", log)
}

func existSqlite() bool {
	conlog("检查sqlite3: ")
	cmd := exec.Command("sqlite3", "--version")
	result, err := cmd.Output()
	if err != nil {
		fmt.Println(alertlog("没有") + "!", result)
		fmt.Println(err)
		return false
	} else {
		fmt.Println(passlog("存在") + "。")
		//fmt.Println(strings.TrimRight(string(result), "\n"))
		return true
	}
}
func sqlite(str string) (string, error) {
	//fmt.Println(str)
	result := ""
	cmd := exec.Command("sqlite3", dbFilePath, str)
	/*stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "p", err
	}
	if err = cmd.Start(); err != nil {
		return "s", err
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		result += scanner.Text()
	}
	if err = cmd.Wait(); err != nil {
		return "w", err
	}
	//fmt.Println(result, str)
	return result, nil*/
	result_b, err := cmd.Output()
	result = strings.TrimSpace(string(result_b))
	result = strings.TrimRight(result, "\n")
	return result, err
}
func saveConfig(table string, key_value map[string]string, id int) error {
	if !validSqlKey(table) {
		return errors.New("\"" + table + "\" is invalid.")
	}
	for key, value := range key_value {
		if !validSqlKey(key) {
			return errors.New("\"" + key + "\" is invalid.")
		}
		if !validSqlValue(value) {
			return errors.New("\"" + value + "\" is invalid.")
		}
	}
	oldvalue, err := readConfig(table, "*", id)
	if err == nil {
		if id == 0 || oldvalue == "" {
			keys := ""
			values := ""
			for key, value := range key_value {
				keys += key + ", "
				if key == "password" {
					value = md5Sum(value)
				}
				values += "\"" + value + "\", "
			}
			keys = keys[0:strings.LastIndex(keys, ",")]
			values = values[0:strings.LastIndex(values, ",")]
			sql := "insert into " + table + " (" + keys + ") values (" + values + ");"
			//fmt.Println(sql)
			_, err = sqlite(sql)
		} else {
			keys := ""
			for key, value := range key_value {
				if key == "password" {
					value = md5Sum(value)
				}
				keys += key + "=\"" + value + "\", "
			}
			keys = keys[0:strings.LastIndex(keys, ",")]
			sql := "update " + table + " set " + keys + " where id=" + strconv.Itoa(id) + ";"
			//fmt.Println(sql)
			_, err = sqlite(sql)
		}
	}
	return err
}
func readConfig(table string, key string, id int) (string, error) {
	if !validSqlKey(table) {
		return "", errors.New("\"" + table + "\" is invalid.")
	}
	if !validSqlKey(key) {
		return "", errors.New("\"" + key + "\" is invalid.")
	}
	if id < 0 {
		return "", errors.New("id is invalid.")
	}
	sql := "select " + key + " from " + table
	if id > 0 {
		sql += " where id=" + strconv.Itoa(id)
	}
	sql += ";"
	//fmt.Println(sql)
	return sqlite(sql)
}
func findConfig(table string, key string, value string) []int {
	var ids []int
	if !validSqlKey(table) {
		ids = append(ids, -1)
		return ids
	}
	if !validSqlKey(key) {
		ids = append(ids, -1)
		return ids
	}
	if !validSqlValue(value) {
		ids = append(ids, -1)
		return ids
	}
	sql := "select id from " + table + " where " + key + "=\"" + value + "\";"
	id_string, err := sqlite(sql)
	if err != nil || id_string == "" {
		ids = append(ids, -1)
		return ids
	}
	id_arr := strSplitLine(id_string)
	for _, id1 := range id_arr {
		id, err := strconv.Atoi(id1)
		if err != nil {
			var ids1 []int
			ids1 = append(ids1, -1)
			return ids1
		} else {
			ids = append(ids, id)
		}
	}
	return ids
}
func delConfig(table string, id int) error {
	sql := "delete from " + table + " where id=" + fmt.Sprint(id) + ";"
	//fmt.Println(sql)
	_, err := sqlite(sql)
	return err
}
func validSqlKey(str string) bool {
	if str == "" {
		return false
	}
	tmp := strings.Index(str, " ")
	if tmp > -1 {
		return false
	}
	tmp = strings.Index(str, ";")
	if tmp > -1 {
		return false
	}
	return validSqlValue(str)
}
func validSqlValue(str string) bool {
	tmp := strings.Index(str, "\"")
	if tmp > -1 {
		return false
	}
	tmp = strings.Index(str, "'")
	if tmp > -1 {
		return false
	}
	return true
}

func removeStrbefor(text string, pre string) string {
	for strings.Index(text, pre) > -1 {
		text = text[(strings.Index(text, pre) + 1):]
	}
	return text
}
func readValueInString(text string, key string) string {
	key = "\"" + key + "\""
	if strings.Index(text, key) > -1 {
		value := text[(strings.Index(text, key) + len(key)):]
		if strings.Index(value, ",") > -1 {
			if strings.Index(value, ",") < strings.Index(value, "\"") {
				value = value[0:strings.Index(value, ",")]
				value = removeStrbefor(value, " ")
				value = removeStrbefor(value, ":")
			} else {
				value = value[(strings.Index(value, "\"") + 1):]
				value = value[0:strings.Index(value, "\"")]
			}
		} else {
			if strings.Index(value, "\"") > -1 {
				value = value[(strings.Index(value, "\"") + 1):]
				value = value[0:strings.Index(value, "\"")]
			} else {
				value = value[0:strings.Index(value, "}")]
				if strings.Index(value, "\n") > -1 {
					value = value[0:strings.Index(value, "\n")]
				}
				value = removeStrbefor(value, " ")
				value = removeStrbefor(value, ":")
			}
		}
		return value
	}
	return ""
}
func wizAport() int {
	sum := 1
	nums := []byte(programName)
	for _, num := range nums {
		sum *= int(num)
	}
	sum = 50000 + sum % 10000
	return sum
}
func startSrv(port int) {
	http.HandleFunc("/", route_web)
	Server := listenHttp("", port)
	defer stopSrv(Server)
	if Server == nil {
		conlog("网页服务启动失败\n")
	} else {
		conlog("网页服务启动成功\n")
		showListening(port)
		wait_Web()
	}
}
func startTmpSrv(port int) {
	http.HandleFunc("/", route_admin)
	Server := listenHttp("", port)
	defer stopSrv(Server)
	if Server == nil {
		conlog("管理服务页面启动失败\n")
	} else {
		conlog("管理服务页面启动成功\n")
		showListening(port)
		wait_Admin(10 * 60)
	}
}
func showListening(port int) {
	conlog("请在浏览器中打开其中一个网址：\n")
	for _, ip := range getLocalIPS() {
		if strings.Index(ip, ":") > -1 { // ipv6加上中括号
			ip = "[" + ip + "]"
		}
		fmt.Printf("     http://%s:%d/\n", ip, port)
	}
}
func getLocalIPS() []string {
	var ips []string
	interfaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}

	for _, inter := range interfaces {
		addrs, err := inter.Addrs()
		if err != nil {
			panic(err)
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			if ipNet.IP.IsLinkLocalUnicast() || ipNet.IP.IsLoopback() {
				continue
			}

			ips = append(ips, ipNet.IP.String())
		}
	}
	//fmt.Println(ips)
	return ips
}
func listenHttp(bindIP string, port int) *http.Server {
	conlog(fmt.Sprintf("正在 %v:%d 启动网站……\n", bindIP, port))
	srv := &http.Server{Addr: fmt.Sprintf("%v:%d", bindIP, port), Handler: nil}
	//srv1, err := net.Listen("tcp", fmt.Sprintf("%v:%d", bindIP, port))
	srv1, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		conlog(fmt.Sprint(err, "\n"))
		conlog(alertlog("网页服务启动失败") + "\n")
		return nil
	}
	go srv.Serve(srv1)
	return srv
}
func stopSrv(Server *http.Server) {
	if Server != nil {
		// 不强制关闭，会等当前会话结束 // elegant graceful
		err := Server.Shutdown(context.Background())
		if err != nil {
			conlog(fmt.Sprint(err, "\n"))
		}
		err = Server.Close()
		if err != nil {
			conlog(fmt.Sprint(err, "\n"))
		} else {
			conlog("网站关闭。\n")
		}
	}
}
func route_web(w http.ResponseWriter, r *http.Request) {
	clearCurrentLine("")
	defer r.Body.Close()
	r.ParseForm()
	path := r.URL.Path
	if path == "/favicon.ico" {
		htmlOutput(w, "", 404, nil)
		return
	}
    query := r.URL.Query()
	data := r.Form

	if checkUserShowLoginPage(w, r) {
		//fmt.Println(data)
		webcookie1, _ := r.Cookie("user")
		webcookie2 := webcookie1.Value
		username := webcookie2[0:strings.Index(webcookie2, ":")]
		userid := findConfig("user", "username", username)[0]
		keys := "needResetPassword,accountDisable,enableSign,workWeiBotKey,dingDingBotToken,SCTKey,SC3Key,cookieIDs"
		result, _ := readConfig("user", keys, userid)
		key_arr := strings.Split(keys, ",")
		res_arr := strings.Split(result, "|")
		user := make(map[string]string)
		for i:=0;i<len(key_arr);i++ {
			user[key_arr[i]] = res_arr[i]
		}
		if user["accountDisable"] == "1" {
			htmlOutput(w, "账号被禁", 401, nil)
			return
		}
		if path == "/" {
			if query.Get("modify") == "password" {
				html :=`<title>修改密码</title>`
				if r.Method == "POST" && data.Get("oldpass") != "" && data.Get("newPass") != "" && data.Get("newPass") == data.Get("pass1") {
					if checkUserPassword(userid, data.Get("oldpass")) {
						values := make(map[string]string)
						values["password"] = data.Get("newPass")
						values["needResetPassword"] = "0"
						err := saveConfig("user", values, userid)
						if err == nil {
							html += `成功<meta http-equiv="refresh" content="2;URL=/">`
							htmlOutput(w, html, 200, nil)
							return
						}
					}
					html += `失败<meta http-equiv="refresh" content="5;URL=/">`
					htmlOutput(w, html, 400, nil)
					return
				}

				html += `
				<h3>修改密码</h3>
<form action="" method="post" name="form1" onsubmit="return check();">
	原密码：<input name="oldpass" type="password"><br>
	新密码：<input name="newPass" type="password"><br>
	再次输入新密码: <input name="pass1" type="password"><br>
	<button>提交</button>
<form>
<script>
	document.form1.oldpass.focus();
	function check() {
		if (document.form1.newPass.value != document.form1.pass1.value) {
			alert("两次密码不一样");
			return false;
		}
		return true;
	}
</script>
				`
				htmlOutput(w, html, 200, nil)
				return
			}
			
			if user["needResetPassword"] == "1" {
				htmlOutput(w, `需要修改密码<meta http-equiv="refresh" content="3;URL=?modify=password">`, 200, nil)
				return
			}

			if query.Get("cookie") == "add" {
				if r.Method == "POST" && data.Get("cookie") != "" {
					//fmt.Println(data)
					cookie1 := strings.ReplaceAll(data.Get("cookie"), "\r", "")
					cookie1 = strings.ReplaceAll(cookie1, "\n", "")
					cookie1 = strings.ReplaceAll(cookie1, "'", "")
					cookie1 = strings.ReplaceAll(cookie1, "\"", "")
					if !checkMHYCookie(data.Get("region"), cookie1) {
						html := `<meta http-equiv="refresh" content="3;URL=/">Cookie不对`
						htmlOutput(w, html, 400, nil)
						return
					}
					values := make(map[string]string)
					values["label"] = data.Get("label")
					values["region"] = data.Get("region")
					values["cookie"] = cookie1
					values["userID"] = fmt.Sprint(userid)
					err := saveConfig("cookie", values, 0)
					if err == nil {
						tmp := findConfig("cookie", "userID", fmt.Sprint(userid))
						if tmp[0] != -1 {
							tmp1 := make([]string, 0)
							for _, v := range tmp {
								tmp1 = append(tmp1, fmt.Sprint(v))
							}
							saveConfig("user", map[string]string {"cookieIDs": strings.Join(tmp1, ",")}, userid)
							html := `保存成功！<br>
<meta http-equiv="refresh" content="2;URL=/">`
							htmlOutput(w, html, 200, nil)
							return
						}
					}
					html := `保存失败：` + fmt.Sprint(err)
					htmlOutput(w, html, 400, nil)
					return
				}

				html := `
<h5>添加cookie</h5>
<form action="" method="post" name="form_cookie_add">
标签（名称）：<input type="text" name="label"><br>
服务器区域：<select name="region">
	<option value="0">米游社（国服）</option>
	<option value="1">Hoyolab（国际服）</option>
</select><br>
Cookie：<br>
<textarea name="cookie" rows="8" cols="48"></textarea><br>
<button name="form_cookie_add">提交</button>
</form>

<a href="https://github.com/qkqpttgf/mhySign" target=_blank>获取Cookie方法</a>`
				htmlOutput(w, html, 200, nil)
				return
			}
			if query.Get("cookie") == "modify" {
				cookieid, err := strconv.Atoi(query.Get("id"))
				if err != nil {
					htmlOutput(w, `id不对<meta http-equiv="refresh" content="2;URL=/">`, 400, nil)
					return
				}
				result, _ := readConfig("cookie", "userID,label,region,cookie", cookieid)
				if result == "" {
					htmlOutput(w, `id不对<meta http-equiv="refresh" content="2;URL=/">`, 400, nil)
					return
				}
				cookie := make(map[string]string)
				tmp := strings.Split(result, "|")
				cookie["userID"] = tmp[0]
				if cookie["userID"] != fmt.Sprint(userid) {
					htmlOutput(w, `id不对<meta http-equiv="refresh" content="2;URL=/">`, 403, nil)
					return
				}
				cookie["label"] = tmp[1]
				cookie["region"] = tmp[2]
				cookie["cookie"] = tmp[3]
				if r.Method == "POST" && data.Get("cookie") != "" {
					//fmt.Println(data)
					cookie1 := strings.ReplaceAll(data.Get("cookie"), "\r", "")
					cookie1 = strings.ReplaceAll(cookie1, "\n", "")
					cookie1 = strings.ReplaceAll(cookie1, "'", "")
					cookie1 = strings.ReplaceAll(cookie1, "\"", "")
					if !checkMHYCookie(data.Get("region"), cookie1) {
						html := `<meta http-equiv="refresh" content="3;URL=/">Cookie不对`
						htmlOutput(w, html, 400, nil)
						return
					}
					values := make(map[string]string)
					values["label"] = data.Get("label")
					values["region"] = data.Get("region")
					values["cookie"] = cookie1
					err := saveConfig("cookie", values, cookieid)
					if err == nil {
						html := `保存成功！<br>
<meta http-equiv="refresh" content="2;URL=/">`
						htmlOutput(w, html, 200, nil)
						return
					}
					html := `保存失败：` + fmt.Sprint(err)
					htmlOutput(w, html, 400, nil)
					return
				}

				html := `
<h5>修改cookie</h5>
<form action="" method="post" name="form_cookie_modify">
标签（名称）：<input type="text" name="label" value="` + cookie["label"] + `"><br>
服务器区域：<select name="region">
	<option value="0"`
				if cookie["region"] != "1" {
					html += " selected"
				}
				html += `>米游社（国服）</option>
	<option value="1"`
				if cookie["region"] == "1" {
					html += " selected"
				}
				html += `>Hoyolab（国际服）</option>
</select><br>
Cookie：<br>
<textarea name="cookie" rows="8" cols="48">` + cookie["cookie"] + `</textarea><br>
<button name="form_cookie_modify">提交</button>
</form>`
				htmlOutput(w, html, 200, nil)
				return
			}
			if query.Get("cookie") == "del" {
				cookieid, err := strconv.Atoi(query.Get("id"))
				if err != nil {
					htmlOutput(w, `id不对<meta http-equiv="refresh" content="2;URL=/">`, 400, nil)
					return
				}
				result, _ := readConfig("cookie", "userID,label,region,cookie", cookieid)
				if result == "" {
					htmlOutput(w, `id不对<meta http-equiv="refresh" content="2;URL=/">`, 400, nil)
					return
				}
				cookie := make(map[string]string)
				tmp := strings.Split(result, "|")
				cookie["userID"] = tmp[0]
				if cookie["userID"] != fmt.Sprint(userid) {
					htmlOutput(w, `id不对<meta http-equiv="refresh" content="2;URL=/">`, 403, nil)
					return
				}
				cookie["label"] = tmp[1]
				cookie["region"] = tmp[2]
				cookie["cookie"] = tmp[3]
				err = delConfig("cookie", cookieid)
				if err == nil {
					html := `删除成功！<br><meta http-equiv="refresh" content="2;URL=/">`
					htmlOutput(w, html, 200, nil)
					return
				}
				html := `<meta http-equiv="refresh" content="2;URL=/">删除失败：` + fmt.Sprint(err)
				htmlOutput(w, html, 400, nil)
				return
			}
			if query.Get("log") == "view" {
				sql := "select * from log where userID=\"" + fmt.Sprint(userid) + "\" order by signTime desc limit 31;"
				result_a, _ := sqlite(sql)
				//fmt.Println(sql, "\n", result_a, "\n", err)
				html := `<title>签到历史记录</title>
<a href="/">返回</a><br>
只显示最近31条：`
				if result_a != "" {
					html += `<table border="1">`
					for _, result_b := range strSplitLine(result_a) {
						if result_b != "" {
							result := strings.Split(result_b, "|")
							html += "<tr>"
							if len(result)>1 {
								html += "<td>" + strings.ReplaceAll(result[2], " ", "<br>") + "</td>"
								html += "<td><pre>" + strings.ReplaceAll(result[3], "\\n", "<br>") + "</pre></td>"
								//html += "<td>" + result[3] + "</td>"
							} else {
								html += "<td>" + result[0] + "</td>"
							}
							html += "</tr>"
						}
					}
					html += `</table>`
				}
				htmlOutput(w, html, 200, nil)
				return
			}

			if r.Method == "POST" {
				if formHasKey(r.PostForm, "form_notify") {
				//if r.PostForm.Has("form_notify") {
					values := make(map[string]string)
					values["workWeiBotKey"] = data.Get("workWeiBotKey")
					values["dingDingBotToken"] = data.Get("dingDingBotToken")
					values["SCTKey"] = data.Get("SCTKey")
					values["SC3Key"] = data.Get("SC3Key")
					err := saveConfig("user", values, userid)
					if err == nil {
						html := `保存成功！<br>
<meta http-equiv="refresh" content="2;URL=` + path + `">`
						htmlOutput(w, html, 200, nil)
						return
					}
					html := `保存失败：` + fmt.Sprint(err)
					htmlOutput(w, html, 400, nil)
					return
				}
			}
			html := `
<title>账号信息 - ` + username + `</title>
用户名：` + username + `（<a href="?modify=password">修改密码</a>）<br><br>
	是否允许签到：`
			if user["enableSign"] == "1" {
				html += "是"
			} else {
				html += "否"
			}
			html += `<br>
<a href="?log=view">查看签到历史</a><br>
<form action="" method="post" name="form_notify" onsubmit="return notifyCheck(this);">
	<h5>通知设置：</h5>
	<a href="https://github.com/qkqpttgf/mhySign" target=_blank>各种key获取方法</a>
	<div>
		企业微信机器人：<br>
		https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=
		<input type="password" name="workWeiBotKey" value="` + user["workWeiBotKey"] + `">
		<br>
	</div>
	<div>
		钉钉机器人：<br>
		https://oapi.dingtalk.com/robot/send?access_token=
		<input type="password" name="dingDingBotToken" value="` + user["dingDingBotToken"] + `">
		<br>
	</div>
	<div>
	<a href="https://sct.ftqq.com/sendkey">方糖Server酱(Turbo版)</a>：<br>
		SendKey：
		<input type="password" name="SCTKey" value="` + user["SCTKey"] + `">
		<br>
	</div>
	<div>
	<a href="https://sc3.ft07.com/sendkey">方糖Server酱(3)</a>：<br>
		SendKey：
		<input type="password" name="SC3Key" value="` + user["SC3Key"] + `">
		<br>
	</div>
	<button name="form_notify">提交</button>
</form>
<br>
<script>
function notifyCheck(e) {
	if (e.SC3Key.value != "") {
		let tmp = e.SC3Key.value;
		if (tmp.substr(0,4) == "sctp") {
			tmp = tmp.substr(4);
			let t = tmp.indexOf("t");
			if (t>0) {
				let n = tmp.substr(0,t)*1;
				if (n>0) return true;
			}
		}
		alert("方糖Server酱3 SendKey 不对");
		return false;
	}
	return true;
}
</script>
<h5>Cookie设置：</h5>
<a href="?cookie=add">添加cookie</a><br>
`
			if len(user["cookieIDs"])>0 {
				html += `
<table border="1">
	<tr><td>id</td><td>标签</td><td>区域</td><td>cookie</td><td>操作</td></tr>`
				for _, cookieid1 := range strings.Split(user["cookieIDs"], ",") {
					if cookieid1 != "" {
						cookieid, _ := strconv.Atoi(cookieid1)
						result, _ := readConfig("cookie", "label,region,cookie", cookieid)
						//fmt.Println(result, err)
						if result != "" {
							res := strings.Split(result, "|")
							html += `
	<tr>
		<td>` + fmt.Sprint(cookieid) + `</td>
		<td>` + res[0] + `</td>
		<td>`
							if res[1] == "1" {
								html += "Hoyolab（国际服）"
							} else {
								html += "米游社（国服）"
							}
							html += `</td>
		<td><input type="text" value="` + res[2] + `" readonly></input></td>
		<td>
			<form action="?cookie=del&id=` + fmt.Sprint(cookieid) + `" method="post" name="form_cookie_del" onsubmit="return confirm('是否要删除` + res[0] + `？')" style="margin: 0">
				<a href="?cookie=modify&id=` + fmt.Sprint(cookieid) + `">修改</a>
				<button name="form_cookie_del">删除</button>
			</form>
		</td>
	</tr>`
						}
					}
				}
				html += `
</table>`
			}
			htmlOutput(w, html, 200, nil)
			return
		}

		head := make(map[string]string)
		head["Location"] = "/"
		htmlOutput(w, "回首页", 302, head)
		return
	}
}
func formHasKey(form url.Values, key string) bool {
	// go 1.17 以下 没有 r.PostForm.Has
	for k, _ := range form {
		//fmt.Println(k, v)
		if k == key {
			return true
		}
	}
	return false
}
func route_admin(w http.ResponseWriter, r *http.Request) {
	quit <- 0
	clearCurrentLine("")
	defer r.Body.Close()
	r.ParseForm()
	path := r.URL.Path
	if path == "/favicon.ico" {
		htmlOutput(w, "", 404, nil)
		return
	}
    query := r.URL.Query()
	data := r.Form
	//fmt.Println(data)

	passid := findConfig("setting", "label", "adminpass")[0]
	if passid < 0 {
	//if pass == "" {
		if data.Get("adminuser") != "" && data.Get("adminpass") != "" && data.Get("adminpass1") != "" && data.Get("adminpass") == data.Get("adminpass1") {
			conlog("  Setting admin\n")
			err := setSetting("adminuser", data.Get("adminuser"))
			err1 := setSetting("adminpass", data.Get("adminpass"))
			if err != nil || err1 != nil {
				conlog(fmt.Sprintln("  Set admin failed\n", err))
				html := `failed
<meta http-equiv="refresh" content="5;URL=">
`
				htmlOutput(w, html, 400, nil)
			} else {
				conlog("  Set admin success\n")
				html := `Success
<meta http-equiv="refresh" content="3;URL=">
`
				htmlOutput(w, html, 201, nil)
			}
		} else {
			conlog("  Set admin first\n")
			html := `
<h5>创建管理员账号</h5>
<form action="" method="post" name="form1" onsubmit="return check();">
	用户名: <input name="adminuser" type="text"><br>
	密码: <input name="adminpass" type="password"><br>
	再次输入密码: <input name="adminpass1" type="password"><br>
	<button>提交</button>
<form>
<script>
	document.form1.adminuser.focus();
	function check() {
		if (document.form1.adminpass.value != document.form1.adminpass1.value) {
			alert("两次密码不一样");
			return false;
		}
		return true;
	}
</script>
`
			htmlOutput(w, html, 201, nil)
		}
		return
	}
	if checkAdminShowLoginPage(w, r) {
		if query.Get("adduser") == "manually" {
			if r.Method == "POST" && data.Get("user") != "" {
				userid := findConfig("user", "username", data.Get("user"))[0]
				if userid > -1 {
					html := `用户` + data.Get("user") + `已经存在！
<meta http-equiv="refresh" content="3;URL=">`
					htmlOutput(w, html, 403, nil)
					return
				} else {
					values := make(map[string]string)
					values["username"] = data.Get("user")
					password := randomPassword()
					values["password"] = password
					values["needResetPassword"] = "1"
					values["enableSign"] = readSetting("enableSign")
					err := saveConfig("user", values, 0)
					if err != nil {
						html := `<a href='/'>返回</a><br>创建失败：` + fmt.Sprint(err)
						htmlOutput(w, html, 400, nil)
					} else {
						html := `<a href='/'>返回</a><br>
						创建成功！<br>
						用户名：` + values["username"] + `<br>
						密码：` + password
						htmlOutput(w, html, 200, nil)
					}
					return
				}
			}
			html := `
创建用户账号：
<form action="" method="post" name="form1">
	用户名: <input name="user" type="text"><br>
	密码将由程序随机生成<br>
	<button>提交</button>
<form>
<script>
	document.form1.user.focus();
</script>
`
			htmlOutput(w, html, 200, nil)
			return
		}
		if path == "/" {
			if r.Method == "POST" {
				if formHasKey(r.PostForm, "form_modify_admin") {
					// 修改密码
					if data.Get("adminuser") != "" && data.Get("adminpass_old") != "" && data.Get("adminpass_new") != "" && data.Get("adminpass_new") == data.Get("adminpass_new1") {
						if md5Sum(data.Get("adminpass_old")) == readSetting("adminpass") {
							err := setSetting("adminuser", data.Get("adminuser"))
							err1 := setSetting("adminpass", data.Get("adminpass_new"))
							if err == nil || err1 == nil {
								html := `成功<meta http-equiv="refresh" content="1;URL=">`
								htmlOutput(w, html, 200, nil)
								return
							}
						}
					}
					htmlOutput(w, "修改失败", 400, nil)
					return
				}
				if formHasKey(r.PostForm, "form_setting") {
				//if r.PostForm.Has("form_setting") {
					// 设定
					for k, v := range data {
						if k != "form_setting" {
							err := setSetting(k, v[0])
							if err != nil {
								htmlOutput(w, "<a href=''>返回</a><br>保存失败：" + fmt.Sprintln(err), 400, nil)
								return
							}
						}
					}
					html := `成功<meta http-equiv="refresh" content="3;URL=">`
					htmlOutput(w, html, 200, nil)
					return
				}
				if formHasKey(r.PostForm, "user_set") {
				//if r.PostForm.Has("user_set") {
					id, _ := strconv.Atoi(data.Get("id"))
					//fmt.Println("set", id)
					values := make(map[string]string)
					if formHasKey(r.PostForm, "username") {
					//if r.PostForm.Has("username") {
						values["username"] = data.Get("username")
					}
					if formHasKey(r.PostForm, "accountDisable") {
					//if r.PostForm.Has("accountDisable") {
						values["accountDisable"] = data.Get("accountDisable")
					}
					if formHasKey(r.PostForm, "enableSign") {
					//if r.PostForm.Has("enableSign") {
						values["enableSign"] = data.Get("enableSign")
					}
					err := saveConfig("user", values, id)
					if err != nil {
						html := `<a href=''>返回</a><br>保存失败：` + fmt.Sprint(err)
						htmlOutput(w, html, 400, nil)
					} else {
						html := `<meta http-equiv="refresh" content="3;URL=">保存成功！`
						htmlOutput(w, html, 200, nil)
					}
					return
				}
				if formHasKey(r.PostForm, "user_reset") {
				//if r.PostForm.Has("user_reset") {
					id, _ := strconv.Atoi(data.Get("id"))
					//fmt.Println("reset", id)
					values := make(map[string]string)
					password := randomPassword()
					values["password"] = password
					values["needResetPassword"] = "1"
					err := saveConfig("user", values, id)
					if err != nil {
						html := `
<meta http-equiv="refresh" content="3;URL="
重置失败：` + fmt.Sprint(err)
						htmlOutput(w, html, 400, nil)
					} else {
						username, _ := readConfig("user", "username", id)
						html := `<a href=''>返回</a><br>
重置成功！<br>
用户名：` + username + `<br>
密码：` + password
						htmlOutput(w, html, 200, nil)
					}
					return
				}
			}

			adminuser := readSetting("adminuser")
			enableRegist := readSetting("enableRegist")
			enableSign := readSetting("enableSign")
			html := `
<title>管理</title>
<form action="" method="post" name="form_modify_admin" onsubmit="return check();">
	<h4>修改管理员用户名与密码：</h4>
	用户名：<input type="text" name="adminuser" value="` + adminuser + `"><br>
	旧密码：<input type="password" name="adminpass_old" value=""><br>
	新密码：<input type="password" name="adminpass_new" value=""><br>
	再次输入新密码：<input type="password" name="adminpass_new1" value=""><br>
	<button name="form_modify_admin">提交</button>
</form>
<script>
	function check() {
		if (document.form_modify_admin.adminpass_new.value != document.form_modify_admin.adminpass_new1.value) {
			alert("两次密码不一样");
			return false;
		}
		return true;
	}
</script>
<form action="" method="post" name="form_setting">
<h4>设定：</h4>
<table>
	<tr>
		<td>游客能自行注册用户：</td>
		<td>
			<label><input type="radio" name="enableRegist" value="0"`
			if enableRegist != "1" {
				html += " checked"
			}
			html += `>否</label>
			<label><input type="radio" name="enableRegist" value="1"`
			if enableRegist == "1" {
				html += " checked"
			}
			html += `>是</label>
		</td>
	</tr>
	<tr>
		<td>默认新用户可以签到：</td>
		<td>
			<label><input type="radio" name="enableSign" value="0"`
			if enableSign != "1" {
				html += " checked"
			}
			html += `>否</label>
			<label><input type="radio" name="enableSign" value="1"`
			if enableSign == "1" {
				html += " checked"
			}
			html += `>是</label>
		</td>
	</tr>
</table>
<button name="form_setting">提交</button>
</form>
`
			html += `
<h4>用户：</h4>
<a href="?adduser=manually">手动添加用户</a>
<table border="1">
	<tr><td>id</td><td>用户名</td><td>Cookie ID</td><td>禁用账号</td><td>可以签到</td><td>操作</td></tr>
`
			keys := "id,username,cookieIDs,accountDisable,enableSign"
			result, _ := readConfig("user", keys, 0)
			for _, line := range strSplitLine(result) {
				if line == "" {
					continue
				}
				userinfo := strings.Split(line, "|")
				html += `
	<form action="" method="post" name="form_user` + userinfo[0] + `">
	<tr>`
				for i, v := range strings.Split(keys, ",") {
					html += `
		<td>`
					if v == "id" {
						html += "<input type=\"text\" size=\"2\" name=\"" + v + "\" value=\"" + userinfo[i] + "\" readonly>"
					}
					if v == "username" {
						html += "<input type=\"text\" name=\"" + v + "\" value=\"" + userinfo[i] + "\">"
					}
					if v == "cookieIDs" {
						html += userinfo[i]
					}
					if v == "accountDisable" {
						html += "<input type=\"hidden\" name=\"" + v + "\" value=\"" + userinfo[i] + "\">"
						html += "<input type=\"checkbox\" onclick=\"this.parentNode.children[0].value = this.checked?'1':'0';\""
						if userinfo[i] == "1" {
							html += " checked"
						}
						html += ">"
					}
					if v == "enableSign" {
						html += "<input type=\"hidden\" name=\"" + v + "\" value=\"" + userinfo[i] + "\">"
						html += "<input type=\"checkbox\" onclick=\"this.parentNode.children[0].value = this.checked?'1':'0';\""
						if userinfo[i] == "1" {
							html += " checked"
						}
						html += ">"
					}
					html += `</td>`
				}
				//for i, v := range userinfo {
				//	html += "<td>" + v + "</td>"
				//}
				html += `
		<td>
			<button name="user_set">提交修改</button>
			<button name="user_reset">重置密码</button>
		</td>`
				html += `
	</tr>
	</form>`
			}
			html += `
</table>`
			htmlOutput(w, html, 200, nil)
			return
		}

		head := make(map[string]string)
		head["Location"] = "/"
		htmlOutput(w, "回首页", 302, head)
		return
	}
}
func readSetting(key string) string {
	id := findConfig("setting", "label", key)[0]
	if id < 0 {
		return ""
	}
	value, _ := readConfig("setting", "setting", id)
	return value
}
func setSetting(key string, value string) error {
	if key == "adminpass" {
		value = md5Sum(value)
	}
	id := findConfig("setting", "label", key)[0]
	if id < 0 {
		sql := "insert into setting (label, setting) values (\"" + key + "\", \"" + value + "\");"
		//fmt.Println(sql)
		_, err := sqlite(sql)
		return err
	} else {
		sql := "update setting set setting=\"" + value + "\" where id=" + strconv.Itoa(id) + ";"
		//fmt.Println(sql)
		_, err := sqlite(sql)
		return err
	}
}
func checkUserPassword(userid int, pass string) bool {
	if userid < 1 {
		return false
	}
	pass1, _ := readConfig("user", "password", userid)
	return md5Sum(pass) == pass1
}
func checkCookie(user string) bool {
	pos1 := strings.Index(user, ":")
	if pos1 < 0 {
		return false
	}
	pos2 := strings.Index(user, "@")
	if pos2 < 0 {
		return false
	}
	username := user[0:pos1]
	userid := findConfig("user", "username", username)[0]
	if userid < 0 {
		return false
	}
	md51 := user[pos1+1:pos2]
	t, _ := strconv.Atoi(user[pos2+1:])
	nt := int(time.Now().Unix())
	if t < nt {
		return false
	}
	pass, _ := readConfig("user", "password", userid)

	return passHashCookie(username, pass, t) == md51
}
func checkAdminCookie(admin string) bool {
	pos1 := strings.Index(admin, ":")
	if pos1 < 0 {
		return false
	}
	pos2 := strings.Index(admin, "@")
	if pos2 < 0 {
		return false
	}
	user := admin[0:pos1]
	adminuser := readSetting("adminuser")
	if user != adminuser {
		return false
	}
	md51 := admin[pos1+1:pos2]
	t, _ := strconv.Atoi(admin[pos2+1:])
	nt := int(time.Now().Unix())
	if t < nt {
		return false
	}
	adminpass := readSetting("adminpass")

	return passHashCookie(user, adminpass, t) == md51
}
func passHashCookie(u string, p string, t int) string {
	return md5Sum(u + "@" + p + "(" + fmt.Sprint(t))
}
func md5Sum(a string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(a)))
}
func randomPassword() string {
	rand.Seed(time.Now().Unix())
	upcaseNum := rand.Intn(3)+1
	lowercaseNum := rand.Intn(3)+1
	numberNum := 8 - upcaseNum - lowercaseNum
	result := ""
	for i:=0;i<upcaseNum;i++ {
		num := rand.Intn(26)+65
		result += string(num)
	}
	for i:=0;i<lowercaseNum;i++ {
		num := rand.Intn(26)+97
		result += string(num)
	}
	for i:=0;i<numberNum;i++ {
		num := rand.Intn(10)+48
		result += string(num)
	}
	return result
}
func checkUserShowLoginPage(w http.ResponseWriter, r *http.Request) bool {
	r.ParseForm()
	//fmt.Println(r)
	path := r.URL.Path
    query := r.URL.Query()
	data := r.Form

	html := ""
	cookie1, err := r.Cookie("user")
	cookie := ""
	if err != nil {
		cookie = ""
	} else {
		cookie = cookie1.Value
	}
	if cookie != "" {
		if checkCookie(cookie) {
			return true
		}
	}
	if query.Get("adduser") == "regist" {
		if readSetting("enableRegist") != "1" {
			htmlOutput(w, `请通知管理员在管理界面允许游客注册！
<meta http-equiv="refresh" content="4;URL=/">`, 401, nil)
			return false
		}
		if r.Method == "POST" && data.Get("user") != "" && data.Get("pass") != "" && data.Get("pass1") != "" && data.Get("pass") == data.Get("pass1") {
			userid := findConfig("user", "username", data.Get("user"))[0]
			if userid > -1 {
				html := `用户` + data.Get("user") + `已经存在！
				<meta http-equiv="refresh" content="5;URL=">`
				htmlOutput(w, html, 403, nil)
			} else {
				values := make(map[string]string)
				values["username"] = data.Get("user")
				values["password"] = data.Get("pass")
				values["enableSign"] = readSetting("enableSign")
				err := saveConfig("user", values, 0)
				if err != nil {
					html := `<meta http-equiv="refresh" content="5;URL=">
创建失败：` + fmt.Sprint(err)
					htmlOutput(w, html, 400, nil)
				} else {
					html := `注册成功！
	<meta http-equiv="refresh" content="2;URL=/">`
					htmlOutput(w, html, 200, nil)
				}
			}
			return false
		} else {
			html := `
<title>注册用户</title>
注册账号：
<form action="" method="post" name="form1" onsubmit="return check();">
	用户名: <input name="user" type="text"><br>
	密码: <input name="pass" type="password"><br>
	再次输入密码: <input name="pass1" type="password"><br>
	<button>提交</button>
<form>
<script>
	document.form1.user.focus();
	function check() {
		if (document.form1.pass.value != document.form1.pass1.value) {
			alert("两次密码不一样");
			return false;
		}
		return true;
	}
</script>
`
			htmlOutput(w, html, 201, nil)
			return false
		}
	}
	if r.Method == "POST" && data.Get("user") != "" && data.Get("pass") != "" {
		userid := findConfig("user", "username", data.Get("user"))[0]
		if userid > -1 {
			accountDisable, _ := readConfig("user", "accountDisable", userid)
			if accountDisable == "1" {
				htmlOutput(w, "账号被禁", 401, nil)
				return false
			}
			pass, _ := readConfig("user", "password", userid)
			if md5Sum(data.Get("pass")) == pass {
				t := int(time.Now().Unix() + 24 * 60 * 60)
				md5 := passHashCookie(data.Get("user"), md5Sum(data.Get("pass")), t)
				html = `
<meta http-equiv="refresh" content="3;URL=` + path + `">
登录成功
<script>
	var expd = new Date();
	expd.setTime(` + fmt.Sprint(t) + `000);
	var expires = "expires=" + expd.toGMTString();
	document.cookie="user=` + data.Get("user") + `:` + md5 + `@` + fmt.Sprint(t) + `; " + expires;
</script>
`
				htmlOutput(w, html, 200, nil)
				conlog(data.Get("user") + "(" + fmt.Sprint(userid) + ") login Success\n")
				return false
			}
		}
		html = `<meta http-equiv="refresh" content="3;URL=` + path + `">登录失败`
		htmlOutput(w, html, 403, nil)
		conlog(data.Get("user") + "(" + fmt.Sprint(userid) + ") login Failed\n")
		return false
	}

	//conlog("Login page\n")
	html += "<title>登录</title>"
	if readSetting("enableRegist") == "1" {
		html += `<a href="?adduser=regist">注册</a><br>`
	}
	html += `登录
<form action="" method="post" name="form1">
	用户名: <input name="user" type="text"><br>
	密码: <input name="pass" type="password"><br>
	<button>提交</button>
<form>
<script>
	document.form1.user.focus();
</script>`
	htmlOutput(w, html, 401, nil)
	return false
}
func checkAdminShowLoginPage(w http.ResponseWriter, r *http.Request) bool {
	r.ParseForm()
	//fmt.Println(r)
	path := r.URL.Path
	data := r.Form

	html := ""
	admincookie1, err := r.Cookie("admin")
	admincookie := ""
	if err != nil {
		admincookie = ""
	} else {
		admincookie = admincookie1.Value
	}
	if admincookie != "" {
		if checkAdminCookie(admincookie) {
			return true
		}
	}
	if r.Method == "POST" && data.Get("adminuser") != "" && data.Get("adminpass") != "" {
		adminpass := readSetting("adminpass")
		if adminpass != "" {
			adminuser := readSetting("adminuser")
			if data.Get("adminuser") == adminuser && md5Sum(data.Get("adminpass")) == adminpass {
				t := int(time.Now().Unix() + 24 * 60 * 60)
				md5 := passHashCookie(data.Get("adminuser"), md5Sum(data.Get("adminpass")), t)
				html = `
<meta http-equiv="refresh" content="3;URL=` + path + `">
登录成功
<script>
	var expd = new Date();
	expd.setTime(` + fmt.Sprint(t) + `000);
	var expires = "expires=" + expd.toGMTString();
	document.cookie="admin=` + data.Get("adminuser") + `:` + md5 + `@` + fmt.Sprint(t) + `; " + expires;
</script>
`
				htmlOutput(w, html, 200, nil)
				conlog("Admin login Success\n")
				return false
			}
		}
		html = `<meta http-equiv="refresh" content="3;URL=` + path + `">登录失败`
		htmlOutput(w, html, 403, nil)
		conlog("Admin login failed\n")
		return false
	} else {
		conlog("Login page\n")
		html = `<h3>登录</h3>
<form action="" method="post" name="form1">
	用户名: <input name="adminuser" type="text"><br>
	密码: <input name="adminpass" type="password"><br>
	<button>提交</button>
<form>
<script>
	document.form1.adminuser.focus();
</script>`
		htmlOutput(w, html, 401, nil)
	}
	return false
}
func htmlOutput(w http.ResponseWriter, body string, code int, head map[string]string) {
	if head == nil {
		head = make(map[string]string)
	}
	_, ok := head["Content-Type"]
	if !ok {
		head["Content-Type"] = "text/html"
	}
	if strings.Index(head["Content-Type"], "text/html")>-1 {
		body = `
<meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
` + body
	}
	head["Server"] = programName + "/" + programVersion + " (" + programAuthor + ")"

	//w.Header().Set("Content-Type", head["Content-Type"])
	for k, v := range head {
		w.Header().Add(k, v)
	}
	w.WriteHeader(code)
	w.Write([]byte(body))
}
type HttpResult struct {
	StatusCode int
	Header http.Header
	Body string
}
func curl(method string, url string, data string, header map[string]string) (HttpResult, error) {
	var result HttpResult
	var err error
	//fmt.Println("初始", result.StatusCode)
	if len(url) < 7 || ( url[0:7] != "http://" && url[0:8] != "https://" ) {
		url = "http://" + url
	}
	var req *http.Request
	req, err = http.NewRequest(method, url, strings.NewReader(data))
	if err != nil {
		fmt.Println(err)
	} else {
		if header == nil {
			header = make(map[string]string)
		}
		if _, ok := header["User-Agent"]; !ok {
			header["User-Agent"] = programName + "/" + programVersion + " (" + programAuthor + ")"
		}
		for k, v := range header {
			req.Header.Add(k, v)
		}
		client := &http.Client{}
		var res *http.Response
		res, err = client.Do(req)
		if err != nil {
			fmt.Println(err)
		} else {
			//fmt.Println(res.StatusCode)
			//fmt.Println(res.Header)
			//fmt.Println(res.Body)
			result.StatusCode = res.StatusCode
			result.Header = res.Header
			var body []byte
			body, err = ioutil.ReadAll(res.Body)
			if err != nil {
				fmt.Println(err)
			} else {
				//fmt.Println(string(body))
				result.Body = string(body)
			}
		}
	}
	return result, err
}
