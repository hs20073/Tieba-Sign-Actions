package main

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/go-github/github"
	jsoniter "github.com/json-iterator/go"
	"golang.org/x/oauth2"
	"io/ioutil"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"log"
	"math"
	"math/rand"
	"net/http"
	url2 "net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var cronType string
var pageSize int = 15

func init() {
	flag.StringVar(&cronType, "cronType", "si", "set config `cronType`")
}

func main() {
	flag.Parse()
	exec()
}
func exec() {
	bdusss := os.Getenv("BDUSS")
	if bdusss == "" {
		log.Println("环境变量必须设置BDUSS")
	}
	bdussArr := strings.Split(bdusss, "\n")
	userList := []User{}
	if cronType == "si" {
		rs := []SignTable{}
		sts := make(chan SignTable, 5000)
		Parallelize(5, len(bdussArr), func(piece int) {
			bduss := bdussArr[piece]
			bdussMd5 := StrToMD5(bduss)
			if !CheckBdussValid(bduss) {
				st := SignTable{"", "", bdussMd5, 0, 0, 0, 0, 0, "未签到", "未签到", 0, "", false, time.Now().UnixNano() / 1e6, 0, nil}
				sts <- st
			} else {
				OneBtnToSign(bduss, sts)
			}
		})
		close(sts)
		if isNotify() {
			//将签到结果上传到github仓库
			if os.Getenv("AUTH_AES_KEY") != "" {
				userList = SaveUserList(bdussArr)
			}
			for st := range sts {
				rs = append(rs, st)
				if os.Getenv("AUTH_AES_KEY") != "" {
					WriteSignDetailData(st.PageData, User{int64(st.Total), st.Name, st.Uid, st.HeadUrl, GetUidWithRandom(st.Uid, userList)})
				}
			}
			ms := GenerateSignResult(0, rs, false)
			fmt.Println(ms + "\n")
			//telegram通知
			TelegramNotifyResult(GenerateSignResult(1, rs, false))
			//Server酱通知
			ServerJiangNotify(GenerateSignResult(1, rs, true))
			//通知完成更新通知次数
			pushNotifyCount()
			WriteSignData(rs)
		} else {
			for st := range sts {
				rs = append(rs, st)
			}
			ms := GenerateSignResult(0, rs, false)
			fmt.Println(ms + "\n")
		}

	} else {
		replys := os.Getenv("REPLY")
		if replys != "" {
			var replyInfo []ReplyInfo
			if err := jsoniter.Unmarshal([]byte(replys), &replyInfo); err != nil {
				log.Println("err: ", err)
			}
			for _, ri := range replyInfo {
				profile := GetUserProfile(GetUid(ri.Bduss))
				portrait := jsoniter.Get([]byte(profile), "user").Get("portrait").ToString()
				name := jsoniter.Get([]byte(profile), "user").Get("name").ToString()
				if isBan(portrait) {
					TelegramNotifyResult(name + "[已被封禁]：\n回帖失败")
				} else {
					tbName := ri.TbName
					tid := ri.Tid
					rr := reply(ri.Bduss, GetTbs(ri.Bduss), tid, GetFid(tbName), tbName, RandMsg(), 2)
					TelegramNotifyResult(name + "[正常]：\n" + rr)
				}
			}
		}
	}

}
func SaveUserList(bdussArr []string) []User {
	userList := []User{}
	//从github上删除多余的文件
	ghToken := os.Getenv("GH_TOKEN")
	repoName := GetRepoName(ghToken)
	for _, bduss := range bdussArr {
		uid := GetUid(bduss)
		profile := GetUserProfile(uid)
		name := jsoniter.Get([]byte(profile), "user").Get("name").ToString()
		nameShow := jsoniter.Get([]byte(profile), "user").Get("name_show").ToString()
		cdnUrl := fmt.Sprintf("https://cdn.jsdelivr.net/gh/%s@latest/data/%s-%s.txt", repoName, uid, GetRandomString(4))
		if nameShow != "" {
			name = nameShow
		}
		portrait := jsoniter.Get([]byte(profile), "user").Get("portrait").ToString()
		headUrl := "https://himg.baidu.com/sys/portrait/item/" + strings.Split(portrait, "?")[0]
		userList = append(userList, User{0, name, uid, headUrl, cdnUrl})
	}
	rd, _ := ioutil.ReadDir("data")
	for _, fi := range rd {
		if !fi.IsDir() && fi.Name() != "sign.json" && fi.Name() != "users.txt" && fi.Name() != "nc" {
			if len(ghToken) > 0 {
				deleteFromGithub(ghToken, "data/"+fi.Name())
			} else {
				fmt.Println("没有配置$GH_TOKEN")
			}
		}
	}
	usersJson, _ := jsoniter.MarshalToString(userList)
	usersJson, err := JsAesEncrypt(usersJson, os.Getenv("AUTH_AES_KEY"))
	if err != nil {
		log.Fatal(err)
	}
	//ioutil.WriteFile("data/users.txt", []byte(usersJson), 0666)
	if len(ghToken) > 0 {
		pushToGithub(usersJson, ghToken, "data/users.txt")
	} else {
		fmt.Println("没有配置$GH_TOKEN")
	}
	return userList
}
func isBan(id string) bool {
	r, _ := Fetch(fmt.Sprintf("https://tieba.baidu.com/home/main?id=%s&fr=userbar", id), nil, "", "")
	if strings.Contains(r, "抱歉，您访问的用户已被屏蔽") {
		return true
	}
	return false
}
func OneBtnToSign(bduss string, sts chan SignTable) {
	tbs := GetTbs(bduss)
	likedTbs, err := GetLikedTiebas(bduss, "")
	if err != nil {
		log.Println("err: ", err)
	}
	chs := make(chan ChanSignResult, 5000)
	Parallelize(5, len(likedTbs), func(piece int) {
		tb := likedTbs[piece]
		//签到一个贴吧
		SyncSignTieBa(tb, bduss, tbs, chs)
	})
	close(chs)
	totalCount := len(likedTbs)
	cookieValidCount := 0
	excepCount := 0
	blackCount := 0
	signCount := 0
	bqCount := 0
	supCount := 0
	bdussMd5 := StrToMD5(bduss)
	pageData := []PageDetail{}
	detailData := []SignDetail{}
	var timespan int64
	c := 0
	page := 0
	i := 0
	for signR := range chs {
		i++
		detailData = append(detailData, SignDetail{signR.Name, signR.Avatar, signR.Level_id, signR.Cur_score, signR.Levelup_score, signR.ErrorCode, signR.RetMsg, signR.SignTime})
		c++
		if c == pageSize || i == totalCount {
			p := Paginator(page, pageSize, totalCount)
			p.List = detailData
			pageData = append(pageData, p)
			detailData = []SignDetail{}
			c = 0
		} else if c == 1 {
			page++
		}
		timespan += signR.Timespan
		if signR.ErrorCode == "1" {
			cookieValidCount++
		} else if signR.ErrorCode == "340006" || signR.ErrorCode == "300004" {
			//贴吧目录出问题，加载数据失败2
			excepCount++
		} else if signR.ErrorCode == "340008" {
			//黑名单
			blackCount++
		} else if signR.ErrorCode == "0" || signR.ErrorCode == "160002" || signR.ErrorCode == "199901" {
			//签到成功、已经签到、账号封禁，签到不涨经验
			signCount++
		} else if signR.ErrorCode == "2280007" || signR.ErrorCode == "340011" || signR.ErrorCode == "1989004" {
			//签到服务忙、签到过快、数据加载失败1
			//三种情况需要重签
			bqCount += Bq(signR.Fname, signR.Fid, bduss, tbs)
		}
		if signR.Sup == "已助攻" {
			supCount++
		}
	}
	wk := WenKuSign(bduss)
	zd := WenKuSign(bduss)
	uid := GetUid(bduss)
	profile := GetUserProfile(uid)
	name := jsoniter.Get([]byte(profile), "user").Get("name").ToString()
	nameShow := jsoniter.Get([]byte(profile), "user").Get("name_show").ToString()
	portrait := jsoniter.Get([]byte(profile), "user").Get("portrait").ToString()
	headUrl := "https://himg.baidu.com/sys/portrait/item/" + strings.Split(portrait, "?")[0]
	if nameShow != "" {
		name = nameShow
	}
	st := SignTable{uid, name, bdussMd5, totalCount, signCount, bqCount, excepCount, blackCount, wk, zd, supCount, headUrl, true, time.Now().UnixNano() / 1e6, timespan, pageData}
	sts <- st
}
func GetUidWithRandom(uid string, userList []User) string {
	for _, u := range userList {
		if u.Uid == uid {
			strings.LastIndex(u.CDNDataUrl, "/")
			ss := strings.Split(u.CDNDataUrl, "/")
			return strings.Split(ss[len(ss)-1], ".")[0]
		}
	}
	return uid
}
func SyncSignTieBa(tb LikedTieba, bduss string, tbs string, chs chan ChanSignResult) ChanSignResult {
	signResult := SignOneTieBa(tb.Name, tb.Id, bduss, tbs)
	sup := CelebritySupport(bduss, "", tb.Id, tbs)
	if sup == "已助攻" || sup == "助攻成功" {
		sup = "已助攻"
	} else {
		sup = "未助攻"
	}
	csr := ChanSignResult{tb.Id, tb.Name, sup, signResult.ErrorMsg, signResult, tb}
	//名人堂助攻
	chs <- csr
	return csr
}

type ChanSignResult struct {
	Fid    string `json:"fid"`
	Fname  string `json:"fname"`
	Sup    string `json:"sup"`
	RetMsg string `json:"ret_msg"`
	SignResult
	LikedTieba
}
type PageDetail struct {
	List       []SignDetail `json:"list"`        //list数据
	PageNo     int          `json:"page_no"`     //当前页码
	PageSize   int          `json:"page_size"`   //每页大小
	Pages      []int        `json:"pages"`       //页码
	TotalPages int          `json:"total_pages"` //总页数
	Total      int          `json:"total"`       //总数
	FirstPage  int          `json:"first_page"`
	LastPage   int          `json:"last_page"`
}
type SignDetail struct {
	Name         string `json:"name"`
	Avatar       string `json:"avatar"`
	LevelId      string `json:"level_id"`
	CurScore     string `json:"cur_score"`
	LevelupScore string `json:"levelup_score"`
	ErrorCode    string `json:"error_code"`
	RetMsg       string `json:"ret_msg"`
	SignTime     int64  `json:"sign_time"`
}
type SignDetailResult struct {
	Data []PageDetail `json:"data"`
	User User         `json:"user"`
}
type User struct {
	Total      int64  `json:"total"`
	Name       string `json:"name"`
	Uid        string `json:"uid"`
	HeadUrl    string `json:"head_url"`
	CDNDataUrl string `json:"cdn_data_url"`
}

func TelegramNotifyResult(ms string) {
	token := os.Getenv("TELEGRAM_APITOKEN")
	chectId := os.Getenv("TELEGRAM_CHAT_ID")
	if token == "" || chectId == "" {
		log.Println("[Telegram]通知：关闭")
	} else {
		bot, err := tgbotapi.NewBotAPI(token)
		if err != nil {
			log.Panic(err)
		}
		bot.Debug = false
		chectIdInt64, _ := strconv.ParseInt(chectId, 10, 64)
		//log.Println("Authorized on account %s", bot.Self.UserName)
		msg := tgbotapi.NewMessage(chectIdInt64, ms)
		bot.Send(msg)
		log.Println("[Telegram]通知：成功")
	}
}
func ServerJiangNotify(ms string) {

	SCKEY := os.Getenv("SCKEY")
	if SCKEY == "" {
		log.Println("[Server酱]通知：关闭")
	} else {
		var postData = map[string]interface{}{
			"text": "[T-S-A]签到结果-" + time.Now().Format("20060102"),
			"desp": ms,
		}
		result, error := Fetch(fmt.Sprintf("https://sc.ftqq.com/%s.send", SCKEY), postData, "", "")
		if error == nil && jsoniter.Get([]byte(result), "errno").ToInt() == 0 {
			log.Println("[Server酱]通知：成功")
		}
	}
}

func GenerateSignResult(t int, rs []SignTable, isSJ bool) string {
	newLine := "\n"
	if isSJ {
		newLine += "\n"
	}
	s := "贴吧ID: " + strconv.Itoa(len(rs)) + newLine
	if len(rs) == 1 && t == 0 {
		s = "贴吧ID: " + HideName(rs[0].Name) + newLine
	} else if len(rs) == 1 && t == 1 {
		s = "贴吧ID: " + rs[0].Name + newLine
	}
	total := []string{}
	Signed := []string{}
	Bq := []string{}
	Excep := []string{}
	Black := []string{}
	Support := []string{}
	wk := []string{}
	zd := []string{}
	for i, r := range rs {
		if len(rs) > 1 {
			if t == 0 {
				s += "\t" + strconv.Itoa(i+1) + ". " + HideName(r.Name) + newLine
			} else {
				s += "\t" + strconv.Itoa(i+1) + ". " + r.Name + newLine
			}
		}
		total = append(total, strconv.Itoa(r.Total))
		Signed = append(Signed, strconv.Itoa(r.Signed))
		Bq = append(Bq, strconv.Itoa(r.Bq))
		Excep = append(Excep, strconv.Itoa(r.Excep))
		Black = append(Black, strconv.Itoa(r.Black))
		Support = append(Support, strconv.Itoa(r.Support))
		wk = append(wk, r.Wenku)
		zd = append(zd, r.Zhidao)
	}
	s += "总数:" + strings.Join(total, "‖") + newLine
	s += "已签到:" + strings.Join(Signed, "‖") + newLine
	s += "补签:" + strings.Join(Bq, "‖") + newLine
	s += "异常:" + strings.Join(Excep, "‖") + newLine
	s += "黑名单:" + strings.Join(Black, "‖") + newLine
	s += "名人堂助攻 :" + strings.Join(Support, "‖") + newLine
	s += "文库:" + strings.Join(wk, "‖") + newLine
	s += "知道:" + strings.Join(zd, "‖") + newLine
	if os.Getenv("AUTH_AES_KEY") != "" && os.Getenv("HOME_URL") != "" {
		url := os.Getenv("HOME_URL") + "/tb.html?k=" + os.Getenv("AUTH_AES_KEY")
		/*body, err := Fetch("https://api.d5.nz/api/dwz/url.php?url="+url, nil, "", "")
		if err == nil && jsoniter.Get([]byte(body), "code").ToString() == "200" {
			url = jsoniter.Get([]byte(body), "url").ToString()
		}*/
		s += "签到详情:" + url
	}
	return s
}

//隐藏id部分内容，保护隐私
func HideName(name string) string {
	arr := strings.Split(name, "")
	if len(arr) == 1 {
		return "*"
	} else if len(arr) == 2 {
		return arr[0] + "*"
	} else if len(arr) > 2 {
		rs := arr[0]
		for i := 1; i < len(arr)-1; i++ {
			rs += "*"
		}
		rs += arr[len(arr)-1]
		return rs
	}
	return "-"
}

func Bq(tbName string, fid string, bduss string, tbs string) int {
	time.Sleep(time.Duration(5) * time.Second)
	signR := SignOneTieBa(tbName, fid, bduss, tbs)
	if signR.ErrorCode == "0" || signR.ErrorCode == "160002" || signR.ErrorCode == "199901" {
		//签到成功、已签到、封禁
		return 1
	} else {
		return 0
	}
}

type SignTable struct {
	Uid      string       `json:"uid"`
	Name     string       `json:"name"`
	BdussMd5 string       `json:"bduss_md5"`
	Total    int          `json:"total"`
	Signed   int          `json:"signed"`
	Bq       int          `json:"bq"`
	Excep    int          `json:"excep"`
	Black    int          `json:"black"`
	Wenku    string       `json:"wenku"`
	Zhidao   string       `json:"zhidao"`
	Support  int          `json:"support"`
	HeadUrl  string       `json:"head_url"`
	IsValid  bool         `json:"is_valid"`
	SignTime int64        `json:"sign_time"`
	Timespan int64        `json:"timespan"`
	PageData []PageDetail `json:"page_data"`
}

type SignResult struct {
	ErrorCode    string `json:"error_code"`
	ErrorMsg     string `json:"error_msg,omitempty" gorm:"-"`
	SignTime     int64  `json:"signTime" gorm:"column:signTime"`
	SignPoint    string `json:"sign_point" gorm:"-"`
	CountSignNum string `json:"count_sign_num" gorm:"-"`
	Timespan     int64  `json:"timespan" gorm:"-"`
}
type LikedTieba struct {
	Id            string `json:"id,omitempty" gorm:"-"`
	Name          string `json:"name,,omitempty" gorm:"-"`
	Favo_type     string `json:"favo_type" gorm:"-"`
	Level_id      string `json:"level_id"`
	Level_name    string `json:"level_name"`
	Cur_score     string `json:"cur_score"`
	Levelup_score string `json:"levelup_score"`
	Avatar        string `json:"avatar"`
	Slogan        string `json:"slogan"`
}
type LikedApiRep struct {
	ForumList  ForumList `json:"forum_list"`
	HasMore    string    `json:"has_more"`
	ServerTime string    `json:"server_time"`
	Time       int64     `json:"time"`
	Ctime      int       `json:"ctime"`
	Logid      int       `json:"logid"`
	ErrorCore  string    `json:"error_core"`
}

type ForumList struct {
	NonGconforum []LikedTieba `json:"non-gconforum"`
	Gconforum    []LikedTieba `json:"gconforum"`
}

//获取uid
func GetUid(bduss string) string {
	body, _ := Fetch("http://tieba.baidu.com/i/sys/user_json", nil, bduss, "")
	return jsoniter.Get([]byte(body), "id").ToString()
}

//获取tbs
func GetTbs(bduss string) string {
	body, err := Fetch("http://tieba.baidu.com/dc/common/tbs", nil, bduss, "")
	if err != nil {
		log.Println("err: ", err)
	}
	isLogin := jsoniter.Get([]byte(body), "is_login").ToInt()
	if isLogin == 1 {
		return jsoniter.Get([]byte(body), "tbs").ToString()
	}
	return ""
}

//公共贴吧请求（带cookie）
func Fetch(url string, postData map[string]interface{}, bduss string, stoken string) (string, error) {
	return FetchWithHeaders(url, postData, bduss, stoken, nil)
}

func FetchWithHeaders(url string, postData map[string]interface{}, bduss string, stoken string, headers map[string]string) (string, error) {
	var request *http.Request
	httpClient := &http.Client{}
	if nil == postData {
		request, _ = http.NewRequest("GET", url, nil)
	} else {
		postParams := url2.Values{}
		for key, value := range postData {
			postParams.Set(key, value.(string))
		}
		postDataStr := postParams.Encode()
		postDataBytes := []byte(postDataStr)
		postBytesReader := bytes.NewReader(postDataBytes)
		request, _ = http.NewRequest("POST", url, postBytesReader)
		request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}
	if "" != bduss {
		request.AddCookie(&http.Cookie{Name: "BDUSS", Value: bduss})
	}
	if "" != stoken {
		request.AddCookie(&http.Cookie{Name: "STOKEN", Value: stoken})
	}
	if headers != nil {
		for key, value := range headers {
			request.Header.Add(key, value)
		}
	}
	response, fetchError := httpClient.Do(request)
	if fetchError != nil {
		return "", fetchError
	}
	defer response.Body.Close()
	body, readError := ioutil.ReadAll(response.Body)
	if readError != nil {
		return "", readError
	}
	return string(body), nil
}

//BDUSS有效性检测
func CheckBdussValid(bduss string) bool {
	body, err := Fetch("http://tieba.baidu.com/dc/common/tbs", nil, bduss, "")
	if err != nil {
		log.Println("err: ", err)
	}
	isLogin := jsoniter.Get([]byte(body), "is_login").ToInt()
	if isLogin == 1 {
		return true
	}
	return false
}

//获取用户关注的所有贴吧
func GetLikedTiebas(bduss string, uid string) ([]LikedTieba, error) {
	pn := 0
	if uid == "" {
		uid = "" //获取uid
	}
	likedTiebaList := make([]LikedTieba, 0)
	for {
		pn++
		var postData = map[string]interface{}{
			"_client_version": "6.2.2",
			"is_guest":        "0",
			"page_no":         strconv.Itoa(pn),
		}
		postData["sign"] = DataSign(postData)
		body, err := Fetch("http://c.tieba.baidu.com/c/f/forum/like", postData, bduss, "")
		if err != nil {
			log.Println("err:", err)
		}
		var likedApiRep LikedApiRep
		if err := jsoniter.Unmarshal([]byte(body), &likedApiRep); err != nil {
			log.Println("err: ", err)
			break
		} else {
			for _, likeTb := range likedApiRep.ForumList.Gconforum {
				likedTiebaList = append(likedTiebaList, likeTb)
			}
			for _, likeTb := range likedApiRep.ForumList.NonGconforum {
				likedTiebaList = append(likedTiebaList, likeTb)
			}
			if likedApiRep.HasMore == "0" {
				break
			}
		}
	}
	return likedTiebaList, nil
}

//回帖
func reply(bduss, tbs, tid, fid, tbName, content string, clientType int) string {
	ct := "2"
	if clientType == 0 {
		ct = strconv.FormatInt(RandInt64(1, 4), 10)
	} else {
		ct = strconv.Itoa(clientType)
	}
	var postData = map[string]interface{}{
		"BDUSS":           bduss,
		"_client_type":    ct,
		"_client_version": "11.7.8.1",
		"_phone_imei":     "000000000000000",
		"anonymous":       "1",
		"content":         content,
		"fid":             fid,
		"from":            "1008621x",
		"is_ad":           "0",
		"kw":              tbName,
		"model":           "MI+5",
		"net_type":        "1",
		"new_vcode":       "1",
		"tbs":             tbs,
		"tid":             tid,
		"timestamp":       strconv.FormatInt(time.Now().UnixNano()/1e6, 10),
		"vcode_tag":       "11",
	}
	postData["sign"] = DataSign(postData)
	headers := make(map[string]string)
	headers["User-Agent"] = "bdtb for Android 11.7.8.1"
	headers["Host"] = "c.tieba.baidu.com"
	body, err := FetchWithHeaders("http://c.tieba.baidu.com/c/c/post/add", postData, "", "", headers)
	if err != nil {
		log.Println("err: ", err)
	}
	j := jsoniter.Get([]byte(body))
	if j.Get("error_code").ToString() == "0" {
		pid := j.Get("pid").ToString()
		return "回帖成功：" + fmt.Sprintf("https://tieba.baidu.com/p/%s?fid=%s&pid=%s#%s", tid, fid, pid, pid)
	} else {
		return "回帖失败：" + body
	}
}

type ReplyInfo struct {
	Bduss      string `json:"bduss"`
	Tid        string `json:"tid"`
	TbName     string `json:"tb_name"`
	ClientType int    `json:"client_type"`
}

func RandMsg() string {
	urls := []string{"https://v1.hitokoto.cn/?encode=json", "https://api.forismatic.com/api/1.0/?method=getQuote&format=json&lang=en"}
	RandInt64(0, 1)
	rand.Seed(time.Now().UnixNano())
	i := rand.Intn(2)
	body, err := Fetch(urls[i], nil, "", "")
	if err != nil {
		fmt.Println("err-50: ", err)
	}
	if i == 0 {
		return jsoniter.Get([]byte(body), "hitokoto").ToString()
	} else {
		return jsoniter.Get([]byte(body), "quoteText").ToString()
	}
}

func RandInt64(min, max int64) int64 {
	rand.Seed(time.Now().UnixNano())
	if min >= max || min == 0 || max == 0 {
		return max
	}
	return rand.Int63n(max-min) + min
}

//签到一个贴吧
func SignOneTieBa(tbName string, fid string, bduss string, tbs string) SignResult {
	start := time.Now().UnixNano() / 1e6
	var postData = map[string]interface{}{
		"_client_id":      "03-00-DA-59-05-00-72-96-06-00-01-00-04-00-4C-43-01-00-34-F4-02-00-BC-25-09-00-4E-36",
		"_client_type":    "4",
		"_client_version": "1.2.1.17",
		"_phone_imei":     "540b43b59d21b7a4824e1fd31b08e9a6",
		"fid":             fid,
		"kw":              tbName,
		"net_type":        "3",
		"tbs":             tbs,
	}
	postData["sign"] = DataSign(postData)
	body, err := Fetch("http://c.tieba.baidu.com/c/c/forum/sign", postData, bduss, "")
	if err != nil {
		log.Println("err: ", err)
	}
	errorCode := jsoniter.Get([]byte(body), "error_code").ToString()
	errorMsg := jsoniter.Get([]byte(body), "error_msg").ToString()
	userInfo := jsoniter.Get([]byte(body), "user_info")
	signResult := SignResult{}
	if errorCode == "0" {
		//签到成功
		if userInfo == nil {
			signResult.SignPoint = "0"
			signResult.CountSignNum = "0"
		} else {
			signResult.SignPoint = userInfo.Get("sign_bonus_point").ToString()
			signResult.CountSignNum = userInfo.Get("cont_sign_num").ToString()
		}

		errorMsg = "签到成功"
	}
	signResult.SignTime = time.Now().UnixNano() / 1e6
	signResult.ErrorCode = errorCode
	signResult.ErrorMsg = errorMsg
	span := (time.Now().UnixNano() / 1e6) - start
	signResult.Timespan = span
	return signResult
}

//文库签到
func WenKuSign(bduss string) string {
	headers := make(map[string]string)
	headers["Host"] = "wenku.baidu.com"
	headers["Referer"] = "https://wenku.baidu.com/task/browse/daily"
	headers["User-Agent"] = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4184.0 Safari/537.36"
	body, err := FetchWithHeaders("https://wenku.baidu.com/task/submit/signin", nil, bduss, "", headers)
	if err != nil {
		log.Println("err: ", err)
	}
	errorNo := jsoniter.Get([]byte(body), "error_no").ToString()
	if body != "" && (errorNo != "0" || errorNo != "1") {
		return "已签到"
	}
	return "未签到"
}

//文库签到
func ZhiDaoSign(bduss string) string {
	stokenBody, err1 := FetchWithHeaders("https://zhidao.baidu.com", nil, bduss, "", map[string]string{"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4184.0 Safari/537.36"})
	if err1 != nil {
		log.Println("err: ", err1)
	}
	stoken := GetBetweenStr(stokenBody, `"stoken":"`, `",`)
	stoken = Substr(stoken, 10, 32)
	time := time.Now().UnixNano() / 1e6
	s := strconv.FormatInt(time, 10)
	var postData = map[string]interface{}{
		"cm":     "100509",
		"stoken": stoken,
		"utdata": "52,52,15,5,9,12,9,52,12,4,15,13,17,12,13,5,13," + s,
	}
	body, err := Fetch("http://zhidao.baidu.com/submit/user", postData, bduss, "")
	if err != nil {
		log.Println("err: ", err)
	}
	errorNo := jsoniter.Get([]byte(body), "errorNo").ToString()
	if body != "" && (errorNo != "0" || errorNo != "2") {
		return "已签到"
	}
	return "未签到"
}

//获取用户基本信息
func GetUserProfile(uid string) string {
	var postData = map[string]interface{}{
		"_client_version": "6.1.2",
		"has_plist":       "2",
		"need_post_count": "1",
		"uid":             uid,
	}
	postData["sign"] = DataSign(postData)
	body, err := Fetch("http://c.tieba.baidu.com/c/u/user/profile", postData, "", "")
	if err != nil {
		log.Println("err: ", err)
	}
	return body
}

//根据贴吧名称获取fid
func GetFid(tbName string) string {
	fid := ""
	body := Get("http://tieba.baidu.com/f/commit/share/fnameShareApi?ie=utf-8&fname=" + tbName)
	jsonBody := jsoniter.Get([]byte(body))
	if jsonBody.Get("no").ToInt() == 0 {
		fid = jsonBody.Get("data").Get("fid").ToString()
	}
	return fid
}

//贴吧未开放此功能
//名人堂助攻： 已助攻{"no":2280006,"error":"","data":[]}
//名人堂助攻： 助攻成功{"no":0,"error":"","data":[...]}
//未关注此吧{"no":3110004,"error":"","data":[]}
func CelebritySupport(bduss string, tbName string, fid string, tbs string) string {
	if fid == "" && tbName == "" {
		log.Println("至少包含贴吧名字、FID中的一个")
	} else if fid == "" && tbName != "" {
		fid = GetFid(tbName)
	}
	if tbs == "" {
		tbs = GetTbs(bduss)
	}
	postData := map[string]interface{}{"forum_id": fid, "tbs": tbs}
	body, err := Fetch("http://tieba.baidu.com/celebrity/submit/getForumSupport", postData, bduss, "")
	if err != nil {
		log.Println("err: ", err)
	}
	npcInfo := jsoniter.Get([]byte(body), "data", 0).Get("npc_info")
	if npcInfo.Size() > 0 {
		npcId := npcInfo.Get("npc_id").ToString()
		postData["npc_id"] = npcId
		suportResult, _ := Fetch("http://tieba.baidu.com/celebrity/submit/support", postData, bduss, "")
		no := jsoniter.Get([]byte(suportResult)).Get("no").ToInt()
		if no == 3110004 {
			return "未关注此吧"
		} else if no == 2280006 {
			return "已助攻"
		} else if no == 0 {
			return "助攻成功"
		}
		return suportResult
	}
	return "该贴吧未开放此功能"
}

//贴吧参数sing MD5签名
func DataSign(postData map[string]interface{}) string {
	var keys []string
	for key, _ := range postData {
		keys = append(keys, key)
	}
	sort.Sort(sort.StringSlice(keys))
	sign_str := ""
	for _, key := range keys {
		sign_str += fmt.Sprintf("%s=%s", key, postData[key])
	}
	sign_str += "tiebaclient!!!"
	return StrToMD5(sign_str)
}
func GetBetweenStr(str, start, end string) string {
	n := strings.Index(str, start)
	if n == -1 {
		n = 0
	}
	str = string([]byte(str)[n:])
	m := strings.Index(str, end)
	if m == -1 {
		m = len(str)
	}
	str = string([]byte(str)[:m])
	return str
}
func Between(str, starting, ending string) string {
	s := strings.Index(str, starting)
	if s < 0 {
		return ""
	}
	s += len(starting)
	e := strings.Index(str[s:], ending)
	if e < 0 {
		return ""
	}
	return str[s : s+e]
}

func Substr(str string, start, length int) string {
	rs := []rune(str)
	rl := len(rs)
	end := 0

	if start < 0 {
		start = rl - 1 + start
	}
	end = start + length

	if start > end {
		start, end = end, start
	}
	if start < 0 {
		start = 0
	}
	if start > rl {
		start = rl
	}
	if end < 0 {
		end = 0
	}
	if end > rl {
		end = rl
	}
	return string(rs[start:end])
}
func StrToMD5(str string) string {
	MD5 := md5.New()
	MD5.Write([]byte(str))
	MD5Result := MD5.Sum(nil)
	signValue := make([]byte, 32)
	hex.Encode(signValue, MD5Result)
	return strings.ToUpper(string(signValue))
}

//http get方法
func Get(url string) string {
	res, _ := http.Get(url)
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	return string(body)
}

//将签到数据写入到json文件中
func WriteSignData(rs []SignTable) {
	tuc := 0
	ttc := 0
	tsc := 0
	tvc := 0
	tbec := 0
	tsuc := 0
	for i, st := range rs {
		tuc++
		if st.IsValid == false {
			jsonBlob, _ := ioutil.ReadFile("data/sign.json")
			var old SignData
			if err := jsoniter.Unmarshal(jsonBlob, &old); err != nil {
				fmt.Println("error: ", err)
			}
			item, err := GetByMd5(old.Sts, st.BdussMd5)
			if err != nil && item.Name != "" {
				rs[i].Name = item.Name
				rs[i].Support = item.Support
				rs[i].Zhidao = item.Zhidao
				rs[i].Wenku = item.Wenku
				rs[i].Black = item.Black
				rs[i].Excep = item.Excep
				rs[i].Signed = item.Signed
				rs[i].Bq = item.Bq
				rs[i].Total = item.Total
				rs[i].HeadUrl = item.HeadUrl
				rs[i].SignTime = item.SignTime
			} else {
				//cookie失效并且未查找到记录
			}
			tvc++
		}
		rs[i].Name = HideName(st.Name)
		ttc += st.Total
		tsc += st.Signed + st.Bq
		tbec += st.Black + st.Excep
		tsuc += st.Support
	}
	sd := SignData{rs, tuc, ttc, tsc, tvc, tbec, tsuc}
	signJson, _ := jsoniter.MarshalToString(sd)
	//ioutil.WriteFile("data/sign.json", []byte(signJson), 0666)
	ghToken := os.Getenv("GH_TOKEN")
	if len(ghToken) > 0 {
		pushToGithub(signJson, ghToken, "data/sign.json")
	} else {
		fmt.Println("没有配置$GH_TOKEN")
	}

}
func WriteSignDetailData(pd []PageDetail, user User) {
	os.Remove("data/" + user.Uid + ".txt")
	sdr := SignDetailResult{pd, user}
	csrsJson, _ := jsoniter.MarshalToString(sdr)
	key := os.Getenv("AUTH_AES_KEY")
	ciphertext, err := JsAesEncrypt(csrsJson, key)
	if err != nil {
		log.Fatal(err)
	}
	//ioutil.WriteFile("data/"+user.CDNDataUrl+".txt", []byte(ciphertext), 0666)
	ghToken := os.Getenv("GH_TOKEN")
	if len(ghToken) > 0 {
		pushToGithub(ciphertext, ghToken, "data/"+user.CDNDataUrl+".txt")
	} else {
		fmt.Println("没有配置$GH_TOKEN")
	}
}
func Exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}
func isNotify() bool {
	nc := os.Getenv("NOTIFY_COUNT")
	count := 1
	if nc != "" {
		c, err := strconv.Atoi(nc)
		if err != nil {
			log.Println("$NOTIFY_COUNT应该为数值类型")
		} else {
			count = c
		}
	}
	if !Exists("data/nc") {
		return true
	}
	ncBlob, _ := ioutil.ReadFile("data/nc")
	date := strings.Split(string(ncBlob), ":")[0]
	if len(strings.Split(string(ncBlob), ":")) > 1 {
		notifyedCount, _ := strconv.Atoi(strings.Split(string(ncBlob), ":")[1])
		timelocal, _ := time.LoadLocation("Asia/Shanghai")
		time.Local = timelocal
		curNow := time.Now().Local()
		curDate := curNow.Format("2006-01-02")
		if date != curDate {
			return true
		} else {
			if notifyedCount < count {
				return true
			}
		}
	} else {
		return true
	}

	return false
}
func pushNotifyCount() {
	timelocal, _ := time.LoadLocation("Asia/Shanghai")
	time.Local = timelocal
	curNow := time.Now().Local()
	curDate := curNow.Format("2006-01-02")
	if !Exists("data/nc") && os.Getenv("GH_TOKEN") != "" {
		pushToGithub(curDate+":"+"1", os.Getenv("GH_TOKEN"), "data/nc")
	} else {
		ncBlob, _ := ioutil.ReadFile("data/nc")
		date := strings.Split(string(ncBlob), ":")[0]
		if len(strings.Split(string(ncBlob), ":")) > 1 {
			notifyedCount, _ := strconv.Atoi(strings.Split(string(ncBlob), ":")[1])
			if date != curDate {
				pushToGithub(curDate+":"+"1", os.Getenv("GH_TOKEN"), "data/nc")
			} else {
				notifyedCount++
				pushToGithub(date+":"+strconv.Itoa(notifyedCount), os.Getenv("GH_TOKEN"), "data/nc")
			}
		} else {
			pushToGithub(curDate+":"+"1", os.Getenv("GH_TOKEN"), "data/nc")
		}
	}
}

type SignData struct {
	Sts  []SignTable
	Tuc  int `json:"tuc"`
	Ttc  int `json:"ttc"`
	Tsc  int `json:"tsc"`
	Tvc  int `json:"tvc"`
	Tbec int `json:"tbec"`
	Tsuc int `json:"tsuc"`
}

func GetByMd5(old []SignTable, bdussMd5 string) (*SignTable, error) {
	item := &SignTable{}
	for _, o := range old {
		if o.BdussMd5 == bdussMd5 {
			item = &o
		}
	}
	return item, nil
}

func pushToGithub(data, token, path string) error {
	r := "Tieba-Sign-Actions"
	if data == "" {
		return errors.New("params error")
	}
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	c := "签到完成上传结果数据：" + path
	sha := ""
	content := &github.RepositoryContentFileOptions{
		Message: &c,
		SHA:     &sha,
		Branch:  github.String("master"),
	}
	op := &github.RepositoryContentGetOptions{}
	user, _, _ := client.Users.Get(ctx, "")
	repo, _, _, er := client.Repositories.GetContents(ctx, user.GetLogin(), r, path, op)
	if er != nil || repo == nil {
		log.Println("get github repository error, create "+path, er)
		content.Content = []byte(data)
		content.SHA = nil
		_, _, err := client.Repositories.CreateFile(ctx, user.GetLogin(), r, path, content)
		if err != nil {
			log.Println(err)
			return err
		}
	} else {
		content.SHA = repo.SHA
		content.Content = []byte(data)
		_, _, err := client.Repositories.UpdateFile(ctx, user.GetLogin(), r, path, content)
		if err != nil {
			log.Println(err)
			return err
		}
	}
	return nil
}

func deleteFromGithub(token, path string) error {
	r := "Tieba-Sign-Actions"
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	c := "删除多余数据文件：" + path
	op := &github.RepositoryContentGetOptions{}
	user, _, _ := client.Users.Get(ctx, "")
	repo, _, _, _ := client.Repositories.GetContents(ctx, user.GetLogin(), r, path, op)
	if repo == nil {
		return nil
	}
	cop := &github.RepositoryContentFileOptions{
		Message: &c,
		SHA:     repo.SHA,
		Branch:  github.String("master"),
	}
	_, _, err := client.Repositories.DeleteFile(ctx, user.GetLogin(), r, path, cop)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
func GetRepoName(token string) string {
	r := "Tieba-Sign-Actions"
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	user, _, _ := client.Users.Get(ctx, "")
	return user.GetLogin() + "/" + r
}

type DoWorkPieceFunc func(piece int)

func Parallelize(workers, pieces int, doWorkPiece DoWorkPieceFunc) {
	toProcess := make(chan int, pieces)
	for i := 0; i < pieces; i++ {
		toProcess <- i
	}
	close(toProcess)

	if pieces < workers {
		workers = pieces
	}

	wg := sync.WaitGroup{}
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer utilruntime.HandleCrash()
			defer wg.Done()
			for piece := range toProcess {
				doWorkPiece(piece)
			}
		}()
	}
	wg.Wait()
}

//---------------AES加密
func JsAesDecrypt(hexS, key []byte) ([]byte, error) {
	hexRaw, err := hex.DecodeString(string(hexS))
	if err != nil {
		return nil, err
	}
	if len(key) == 0 {
		return nil, errors.New("key 不能为空")
	}
	pkey := paddingLeft(key, '0', 16)
	block, err := aes.NewCipher(pkey) //选择加密算法
	if err != nil {
		return nil, fmt.Errorf("key 长度必须 16/24/32长度: %s", err)
	}
	blockModel := cipher.NewCBCDecrypter(block, pkey)
	plantText := make([]byte, len(hexRaw))
	blockModel.CryptBlocks(plantText, hexRaw)
	plantText = pkcs7UnPadding(plantText)
	return plantText, nil
}

func paddingLeft(ori []byte, pad byte, length int) []byte {
	if len(ori) >= length {
		return ori[:length]
	}
	pads := bytes.Repeat([]byte{pad}, length-len(ori))
	return append(pads, ori...)
}

func JsAesEncrypt(raw string, key string) (string, error) {
	origData := []byte(raw)
	// 转成字节数组
	if len(key) == 0 {
		return "", errors.New("key 不能为空")
	}
	k := paddingLeft([]byte(key), '0', 16)

	// 分组秘钥
	block, err := aes.NewCipher(k)
	if err != nil {
		return "", fmt.Errorf("填充秘钥key的16位，24,32分别对应AES-128, AES-192, or AES-256  key 长度必须 16/24/32长度: %s", err)
	}
	// 获取秘钥块的长度
	blockSize := block.BlockSize()
	// 补全码
	origData = pkcs7Padding(origData, blockSize)
	// 加密模式
	blockMode := cipher.NewCBCEncrypter(block, k)
	// 创建数组
	cryted := make([]byte, len(origData))
	// 加密
	blockMode.CryptBlocks(cryted, origData)
	//使用RawURLEncoding 不要使用StdEncoding
	//不要使用StdEncoding  放在url参数中会导致错误
	return hex.EncodeToString(cryted), nil
}

func pkcs7UnPadding(plantText []byte) []byte {
	length := len(plantText)
	unpadding := int(plantText[length-1])
	return plantText[:(length - unpadding)]
}

func pkcs7Padding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padtext...)

}

func GetRandomString(l int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyz"
	bytes := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < l; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}

func Paginator(page, pageSize int, total int) PageDetail {
	var firstpage int //前一页地址
	var lastpage int  //后一页地址
	//根据nums总数，和prepage每页数量 生成分页总数
	totalpages := int(math.Ceil(float64(total) / float64(pageSize))) //page总数
	if page > totalpages {
		page = totalpages
	}
	if page <= 0 {
		page = 1
	}
	var pages []int
	switch {
	case page >= totalpages-5 && totalpages > 5: //最后5页
		start := totalpages - 5 + 1
		firstpage = page - 1
		lastpage = int(math.Min(float64(totalpages), float64(page+1)))
		pages = make([]int, 5)
		for i, _ := range pages {
			pages[i] = start + i
		}
	case page >= 3 && totalpages > 5:
		start := page - 3 + 1
		pages = make([]int, 5)
		firstpage = page - 3
		for i, _ := range pages {
			pages[i] = start + i
		}
		firstpage = page - 1
		lastpage = page + 1
	default:
		pages = make([]int, int(math.Min(5, float64(totalpages))))
		for i, _ := range pages {
			pages[i] = i + 1
		}
		firstpage = int(math.Max(float64(1), float64(page-1)))
		lastpage = page + 1
		//fmt.Println(pages)
	}
	totalPages := 0
	if total%pageSize == 0 {
		totalPages = total / pageSize
	} else {
		totalPages = total/pageSize + 1
	}
	pd := PageDetail{}
	pd.Pages = pages
	pd.Total = total
	pd.FirstPage = firstpage
	pd.LastPage = lastpage
	pd.PageNo = page
	pd.TotalPages = totalPages
	return pd
}
