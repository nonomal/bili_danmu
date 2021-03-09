package send

import (
	"strings"
	"strconv"
	"encoding/json"

	c "github.com/qydysky/bili_danmu/CV"
	p "github.com/qydysky/part"
)

//每5s一个令牌，最多等10秒
var danmu_s_limit = p.Limit(1, 5000, 10000)

//弹幕发送
func Danmu_s(msg,Cookie string, roomid int) {
	//等待令牌时阻塞，超时返回true
	if danmu_s_limit.TO() {return}

	l := c.Log.Base("弹幕发送")

	if msg == "" || Cookie == "" || roomid == 0{
		l.L(`E: `,"输入参数不足")
		return
	}
	if i := strings.Index(Cookie, "{"); i != -1 {
		l.L(`E: `,"Cookie格式错误,需为 key=val; key=val 式")
		return
	}

	if i := strings.Index(Cookie, "PVID="); i != -1 {//删除PVID
		if d := strings.Index(Cookie[i:], ";"); d == -1 {
			Cookie = Cookie[:i]
		} else {
			Cookie = Cookie[:i] + Cookie[i + d + 1:]
		}
	}

	csrf,_ := c.Cookie.LoadV(`bili_jct`).(string)
	if csrf == `` {l.L(`E: `,"Cookie错误,无bili_jct=");return}

	PostStr := `color=16777215&fontsize=25&mode=1&msg=` + msg + `&rnd=` + strconv.Itoa(int(p.Sys().GetSTime())) + `&roomid=` + strconv.Itoa(roomid) + `&bubble=0&csrf_token=` + csrf + `&csrf=` + csrf
	l.L(`I: `,"发送", msg, "至", roomid)
	r := p.Req()
	err := r.Reqf(p.Rval{
		Url:"https://api.live.bilibili.com/msg/send",
		PostStr:PostStr,
		Retry:2,
		Timeout:10,
		Header:map[string]string{
			`Host`: `api.live.bilibili.com`,
			`User-Agent`: `Mozilla/5.0 (X11; Linux x86_64; rv:83.0) Gecko/20100101 Firefox/83.0`,
			`Accept`: `application/json, text/javascript, */*; q=0.01`,
			`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
			`Accept-Encoding`: `gzip, deflate, br`,
			`Content-Type`: `application/x-www-form-urlencoded; charset=UTF-8`,
			`Origin`: `https://live.bilibili.com`,
			`Connection`: `keep-alive`,
			`Pragma`: `no-cache`,
			`Cache-Control`: `no-cache`,
			`Referer`:"https://live.bilibili.com/" + strconv.Itoa(roomid),
			`Cookie`:Cookie,
		},
	})
	if err != nil {
		l.L(`E: `,err)
		return
	}
	
	var res struct{
		Code int `json:"code"`
		Message string `json:"message"`
	}

	if e := json.Unmarshal(r.Respon, &res);e != nil{
		l.L(`E: `,e)
	}

	if res.Code != 0 {
		l.L(`E: `, `产生错误：`,res.Code, res.Message)
	}

}