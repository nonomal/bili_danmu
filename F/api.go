package F

import (
	"time"
	"fmt"
	"os"
	"strconv"
	"strings"
    "context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/skratchdot/open-golang/open"
	qr "github.com/skip2/go-qrcode"
	c "github.com/qydysky/bili_danmu/CV"
	web "github.com/qydysky/part/web"
	funcCtrl "github.com/qydysky/part/funcCtrl"
	g "github.com/qydysky/part/get"
	p "github.com/qydysky/part"
	uuid "github.com/gofrs/uuid"
)

type api struct {
	Roomid int
	Uid int
	Url []string
	Live []string
	Live_status float64
	Locked bool
	Token string
	Parent_area_id int
	Area_id int
}

var apilog = c.Log.Base(`api`)
var api_limit = p.Limit(1,2000,30000)//频率限制1次/2s，最大等待时间30s

func New_api(Roomid int) (o *api) {
	apilog.Base_add(`新建`).L(`T: `,"ok")
	o = new(api)
	o.Roomid = Roomid
	o.Parent_area_id = -1
	o.Area_id = -1
	o.Get_info()
	return
}

func (i *api) Get_info() (o *api) {
	o = i
	apilog := apilog.Base_add(`获取房号`).L(`T: `, `开始`)

	if o.Roomid == 0 {
		apilog.L(`E: `,"还未New_api")
		return
	}
	if api_limit.TO() {return}//超额请求阻塞，超时将取消

	defer o.Get_LIVE_BUVID()
	
	Roomid := strconv.Itoa(o.Roomid)

	r := g.Get(p.Rval{
		Url:"https://live.bilibili.com/blanc/" + Roomid,
	})
	//uid
	if tmp := r.S(`"uid":`, `,`, 0, 0);tmp.Err != nil {
		// apilog.L(`E: `,"uid", tmp.Err)
	} else if i,err := strconv.Atoi(tmp.RS[0]); err != nil{
		apilog.L(`E: `,"uid", err)
	} else {
		o.Uid = i
		c.UpUid = i
	}
	//Title
	if e := r.S(`"title":"`, `",`, 0, 0).Err;e == nil {
		c.Title = r.RS[0]
	}
	//主播id
	if e := r.S(`"base_info":{"uname":"`, `",`, 0, 0).Err;e == nil {
		c.Uname = r.RS[0]
	}
	//分区
	if e := r.S(`"parent_area_id":`, `,`, 0, 0).Err;e == nil {
		if tmp,e := strconv.Atoi(r.RS[0]);e != nil{
			apilog.L(`E: `,"parent_area_id", e)
		} else {o.Parent_area_id = tmp}
	}
	if e := r.S(`"area_id":`, `,`, 0, 0).Err;e == nil {
		if tmp,e := strconv.Atoi(r.RS[0]);e != nil{
			apilog.L(`E: `,"area_id", e)
		} else {o.Area_id = tmp}
	}
	//roomid
	if tmp := r.S(`"room_id":`, `,`, 0, 0);tmp.Err != nil {
		// apilog.L(`E: `,"room_id", tmp.Err)
	} else if i,err := strconv.Atoi(tmp.RS[0]); err != nil{
		apilog.L(`E: `,"room_id", err)
	} else {
		o.Roomid = i
	}

	if	o.Area_id != -1 && 
		o.Parent_area_id != -1 &&
		o.Roomid != 0 &&
		o.Uid != 0 &&
		c.Title != ``{return}

	{//使用其他api
		Cookie := make(map[string]string)
		c.Cookie.Range(func(k,v interface{})(bool){
			Cookie[k.(string)] = v.(string)
			return true
		})

		req := p.Req()
		if err := req.Reqf(p.Rval{
			Url:"https://api.live.bilibili.com/xlive/web-room/v1/index/getInfoByRoom?room_id=" + Roomid,
			Header:map[string]string{
				`Referer`:"https://live.bilibili.com/" + Roomid,
				`Cookie`:p.Map_2_Cookies_String(Cookie),
			},
			Timeout:10,
			Retry:2,
		});err != nil {
			apilog.L(`E: `,err)
			return
		}
		var tmp struct{
			Code int `json:"code"`
			Message string `json:"message"`
			Data struct{
				Room_info struct{
					Uid int `json:"uid"`
					Room_id int `json:"room_id"`
					Title string `json:"title"`
					Lock_status int `json:"lock_status"`
					Area_id int `json:"area_id"`
					Parent_area_id int `json:"parent_area_id"`
				} `json:"room_info"`
				Anchor_info struct{
					Base_info struct{
						Uname string `json:"uname"`
					} `json:"base_info"`
				} `json:"anchor_info"`
			} `json:"data"`
		}
		if e := json.Unmarshal(req.Respon, &tmp);e != nil{
			apilog.L(`E: `,e)
			return
		}

		//错误响应
		if tmp.Code != 0 {
			apilog.L(`E: `,`code`,tmp.Message)
			return
		}

		//主播
		if tmp.Data.Anchor_info.Base_info.Uname != `` && c.Uname == ``{
			c.Uname = tmp.Data.Anchor_info.Base_info.Uname
		}

		//主播id
		if tmp.Data.Room_info.Uid != 0{
			o.Uid = tmp.Data.Room_info.Uid
			c.UpUid = tmp.Data.Room_info.Uid
		} else {
			apilog.L(`W: `,"data.room_info.Uid = 0")
			return
		}

		//分区
		if tmp.Data.Room_info.Parent_area_id != 0{
			o.Parent_area_id = tmp.Data.Room_info.Parent_area_id
		} else {
			apilog.L(`W: `,"直播间未设置主分区！")
			return
		}
		if tmp.Data.Room_info.Area_id != 0{
			o.Area_id = tmp.Data.Room_info.Area_id
		} else {
			apilog.L(`W: `,"直播间未设置分区！")
			return
		}

		//房间id
		if tmp.Data.Room_info.Room_id != 0{
			o.Roomid = tmp.Data.Room_info.Room_id
		} else {
			apilog.L(`W: `,"data.room_info.room_id = 0")
			return
		}
		
		//房间标题
		if tmp.Data.Room_info.Title != ``{
			c.Title = tmp.Data.Room_info.Title
		} else {
			apilog.L(`W: `,"直播间无标题")
			return
		}

		//直播间是否被封禁
		if tmp.Data.Room_info.Lock_status == 1{
			apilog.L(`W: `,"直播间封禁中")
			o.Locked = true
			return
		}
	}
	return
}

func (i *api) Get_live(qn ...string) (o *api) {
	o = i
	apilog := apilog.Base_add(`直播流信息`)

	if o.Roomid == 0 {
		apilog.L(`E: `,"还未New_api")
		return
	}
	if api_limit.TO() {return}//超额请求阻塞，超时将取消
	
	CookieM := make(map[string]string)
	c.Cookie.Range(func(k,v interface{})(bool){
		CookieM[k.(string)] = v.(string)
		return true
	})

	Cookie := p.Map_2_Cookies_String(CookieM)
	if i := strings.Index(Cookie, "PVID="); i != -1 {
		if d := strings.Index(Cookie[i:], ";"); d == -1 {
			Cookie = Cookie[:i]
		} else {
			Cookie = Cookie[:i] + Cookie[i + d + 1:]
		}
	} else {
		qn = []string{}
	}

	if len(qn) == 0 || qn[0] == "0" || qn[0] == "" {//html获取
		r := g.Get(p.Rval{
			Url:"https://live.bilibili.com/blanc/" + strconv.Itoa(o.Roomid),
			Header:map[string]string{
				`Cookie`:Cookie,
			},
		})
		if e := r.S(`"durl":[`, `]`, 0, 0).Err;e == nil {
			if urls := p.Json().GetArrayFrom("[" + r.RS[0] + "]", "url");urls != nil {
				apilog.L(`I: `,"直播中")
				c.Liveing = true
				o.Live_status = 1
				for _,v := range urls {
					o.Live = append(o.Live, v.(string))
				}
				return
			}
		}
		if e := r.S(`"live_time":"`, `"`, 0, 0).Err;e == nil {
			c.Live_Start_Time,_ = time.Parse("2006-01-02 15:04:05", r.RS[0])
		}
	}

	cu_qn := "0"
	{//api获取
		req := p.Req()
		if err := req.Reqf(p.Rval{
			Url:"https://api.live.bilibili.com/xlive/web-room/v2/index/getRoomPlayInfo?no_playurl=0&mask=1&qn=0&platform=web&protocol=0,1&format=0,2&codec=0,1&room_id=" + strconv.Itoa(o.Roomid),
			Header:map[string]string{
				`Referer`:"https://live.bilibili.com/" + strconv.Itoa(o.Roomid),
				`Cookie`:Cookie,
			},
			Timeout:10,
			Retry:2,
		});err != nil {
			apilog.L(`E: `,err)
			return
		}
		res := string(req.Respon)
		if code := p.Json().GetValFromS(res, "code");code == nil || code.(float64) != 0 {
			apilog.L(`E: `,"code", code)
			return
		}
		if is_locked := p.Json().GetValFrom(res, "data.is_locked");is_locked == nil {
			apilog.L(`E: `,"data.is_locked", is_locked)
			return
		} else if is_locked.(bool) {
			apilog.L(`W: `,"直播间封禁中")
			o.Locked = true
			return
		}
		if live_status := p.Json().GetValFrom(res, "data.live_status");live_status == nil {
			apilog.L(`E: `,"data.live_status", live_status)
			return
		} else {
			o.Live_status = live_status.(float64)
			switch live_status.(float64) {
			case 2:
				c.Liveing = false
				apilog.L(`I: `,"轮播中")
				return
			case 0: //未直播
				c.Liveing = false
				apilog.L(`I: `,"未在直播")
				return
			case 1:
				c.Liveing = true
				apilog.L(`I: `,"直播中")
			default:
				apilog.L(`W: `,"live_status:", live_status)
			}
		}
		if codec0 := p.Json().GetValFrom(res, "data.playurl_info.playurl.stream.[0].format.[0].codec.[0]");codec0 != nil {//直播流链接
			base_url := p.Json().GetValFrom(codec0, "base_url")
			if base_url == nil {return}
			url_info := p.Json().GetValFrom(codec0, "url_info")
			if v,ok := url_info.([]interface{});!ok || len(v) == 0 {return}
			for _,v := range url_info.([]interface{}) {
				host := p.Json().GetValFrom(v, "host")
				extra := p.Json().GetValFrom(v, "extra")
				if host == nil || extra == nil {continue}
				o.Live = append(o.Live, host.(string) + base_url.(string) + extra.(string))
			}
		}
		if len(o.Live) == 0 {apilog.L(`E: `,"live url is nil");return}

		if i := p.Json().GetValFrom(res, "data.playurl_info.playurl.stream.[0].format.[0].codec.[0].current_qn"); i != nil {
			cu_qn = strconv.Itoa(int(i.(float64)))
		}
		if i := p.Json().GetValFrom(res, "data.live_time"); i != nil {
			c.Live_Start_Time = time.Unix(int64(i.(float64)),0).In(time.FixedZone("UTC-8", -8*60*60))
		}

		if len(qn) != 0 && qn[0] != "0" && qn[0] != "" {
			var (
				accept_qn_request bool
				tmp_qn int
				e error
			)
			if tmp_qn,e = strconv.Atoi(qn[0]);e != nil {apilog.L(`E: `,`qn error`,e);return}
			if i,ok := p.Json().GetValFrom(res, "data.playurl_info.playurl.stream.[0].format.[0].codec.[0].accept_qn").([]interface{}); ok && len(i) != 0 {
				for _,v := range i {
					if o,ok := v.(float64);ok && int(o) == tmp_qn {accept_qn_request = true}
				}
			}
			if !accept_qn_request {
				apilog.L(`E: `,`qn不在accept_qn中`);
				return
			}
			if _,ok := c.Default_qn[qn[0]];!ok{
				apilog.L(`W: `,"清晰度未找到", qn[0], ",使用默认")
				return
			}
			if err := req.Reqf(p.Rval{
				Url:"https://api.live.bilibili.com/xlive/web-room/v2/index/getRoomPlayInfo?no_playurl=0&mask=1&platform=web&protocol=0,1&format=0,2&codec=0,1&room_id=" + strconv.Itoa(o.Roomid) + "&qn=" + qn[0],
				Header:map[string]string{
					`Cookie`:Cookie,
					`Referer`:"https://live.bilibili.com/" + strconv.Itoa(o.Roomid),
				},
				Timeout:10,
				Retry:2,
			});err != nil {
				apilog.L(`E: `,err)
				return
			}
			res = string(req.Respon)
			if codec0 := p.Json().GetValFrom(res, "data.playurl_info.playurl.stream.[0].format.[0].codec.[0]");codec0 != nil {//直播流链接
				base_url := p.Json().GetValFrom(codec0, "base_url")
				if base_url == nil {return}
				url_info := p.Json().GetValFrom(codec0, "url_info")
				if v,ok := url_info.([]interface{});!ok || len(v) == 0 {return}
				for _,v := range url_info.([]interface{}) {
					host := p.Json().GetValFrom(v, "host")
					extra := p.Json().GetValFrom(v, "extra")
					if host == nil || extra == nil {continue}
					o.Live = append(o.Live, host.(string) + base_url.(string) + extra.(string))
				}
			}
			if len(o.Live) == 0 {apilog.L(`E: `,"live url is nil");return}
	
			if i := p.Json().GetValFrom(res, "data.playurl_info.playurl.stream.[0].format.[0].codec.[0].current_qn"); i != nil {
				cu_qn = strconv.Itoa(int(i.(float64)))
			}
		}
	}

	if v,ok := c.Default_qn[cu_qn];ok {
		apilog.L(`I: `,"当前清晰度:", v)
	}
	return
}

func (i *api) Get_host_Token() (o *api) {
	o = i
	apilog := apilog.Base_add(`获取Token`)

	if o.Roomid == 0 {
		apilog.L(`E: `,"还未New_api")
		return
	}
	Roomid := strconv.Itoa(o.Roomid)
	if api_limit.TO() {return}//超额请求阻塞，超时将取消
	
	Cookie := make(map[string]string)
	c.Cookie.Range(func(k,v interface{})(bool){
		Cookie[k.(string)] = v.(string)
		return true
	})

	req := p.Req()
	if err := req.Reqf(p.Rval{
		Url:"https://api.live.bilibili.com/xlive/web-room/v1/index/getDanmuInfo?type=0&id=" + Roomid,
		Header:map[string]string{
			`Referer`:"https://live.bilibili.com/" + Roomid,
			`Cookie`:p.Map_2_Cookies_String(Cookie),
		},
		Timeout:10,
		Retry:2,
	});err != nil {
		apilog.L(`E: `,err)
		return
	}
	res := string(req.Respon)
	if msg := p.Json().GetValFromS(res, "message");msg == nil || msg != "0" {
		apilog.L(`E: `,"message", msg)
		return
	}

	_Token := p.Json().GetValFromS(res, "data.token")
	if _Token == nil {
		apilog.L(`E: `,"data.token", _Token, res)
		return
	}
	o.Token = _Token.(string)

	if host_list := p.Json().GetValFromS(res, "data.host_list");host_list == nil {
		apilog.L(`E: `,"data.host_list", host_list)
		return
	} else {
		for k, v := range host_list.([]interface{}) {
			if _host := p.Json().GetValFrom(v, "host");_host == nil {
				apilog.L(`E: `,"data.host_list[", k, "].host", _host)
				continue
			} else {
				o.Url = append(o.Url, "wss://" + _host.(string) + "/sub")
			}			
		}
		apilog.L(`T: `,"ok")
	}

	return
}

func Get_face_src(uid string) (string) {
	if uid == "" {return ""}
	if api_limit.TO() {return ""}//超额请求阻塞，超时将取消
	apilog := apilog.Base_add(`获取头像`)
	
	Cookie := make(map[string]string)
	c.Cookie.Range(func(k,v interface{})(bool){
		Cookie[k.(string)] = v.(string)
		return true
	})

	req := p.Req()
	if err := req.Reqf(p.Rval{
		Url:"https://api.live.bilibili.com/xlive/web-room/v1/index/getDanmuMedalAnchorInfo?ruid=" + uid,
		Header:map[string]string{
			`Referer`:"https://live.bilibili.com/" + strconv.Itoa(c.Roomid),
			`Cookie`:p.Map_2_Cookies_String(Cookie),
		},
		Timeout:10,
		Retry:2,
	});err != nil {
		apilog.L(`E: `,err)
		return ""
	}
	res := string(req.Respon)
	if msg := p.Json().GetValFromS(res, "message");msg == nil || msg != "0" {
		apilog.L(`E: `,"message", msg)
		return ""
	}

	rface := p.Json().GetValFromS(res, "data.rface")
	if rface == nil {
		apilog.L(`E: `,"data.rface", rface)
		return ""
	}
	return rface.(string) + `@58w_58h`
}

func (i *api) Get_OnlineGoldRank() {
	if i.Uid == 0 || c.Roomid == 0 {
		apilog.Base_add("Get_OnlineGoldRank").L(`E: `,"i.Uid == 0 || c.Roomid == 0")
		return
	}
	if api_limit.TO() {return}//超额请求阻塞，超时将取消
	apilog := apilog.Base_add(`获取贡献榜`)

	var session_roomid = c.Roomid
	var self_loop func(page int)
	self_loop = func(page int){
		if page <= 0 || session_roomid != c.Roomid{return}
		
		Cookie := make(map[string]string)
		c.Cookie.Range(func(k,v interface{})(bool){
			Cookie[k.(string)] = v.(string)
			return true
		})

		req := p.Req()
		if err := req.Reqf(p.Rval{
			Url:`https://api.live.bilibili.com/xlive/general-interface/v1/rank/getOnlineGoldRank?ruid=`+strconv.Itoa(i.Uid)+`&roomId=`+strconv.Itoa(c.Roomid)+`&page=`+strconv.Itoa(page)+`&pageSize=20`,
			Header:map[string]string{
				`Host`: `api.live.bilibili.com`,
				`User-Agent`: `Mozilla/5.0 (X11; Linux x86_64; rv:83.0) Gecko/20100101 Firefox/83.0`,
				`Accept`: `application/json, text/plain, */*`,
				`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
				`Accept-Encoding`: `gzip, deflate, br`,
				`Origin`: `https://live.bilibili.com`,
				`Connection`: `keep-alive`,
				`Pragma`: `no-cache`,
				`Cache-Control`: `no-cache`,
				`Referer`:"https://live.bilibili.com/" + strconv.Itoa(c.Roomid),
				`Cookie`:p.Map_2_Cookies_String(Cookie),
			},
			Timeout:3,
			Retry:2,
		});err != nil {
			apilog.L(`E: `,err)
			return
		}
		res := string(req.Respon)
		if msg := p.Json().GetValFromS(res, "message");msg == nil || msg != "0" {
			apilog.L(`E: `,"message", msg)
			return
		}
		if onlineNum := p.Json().GetValFromS(res, "data.onlineNum");onlineNum == nil {
			apilog.L(`E: `,"onlineNum", onlineNum)
			return
		} else {
			tmp_onlineNum := onlineNum.(float64)
			if tmp_onlineNum == 0 {
				return
			}

			var score = 0.0
			if tmp_score_list := p.Json().GetArrayFrom(p.Json().GetValFromS(res, "data.OnlineRankItem"), "score");len(tmp_score_list) != 0 {
				for _,v := range tmp_score_list {
					score += v.(float64)/10
				}
			}
			//传入消息队列
			c.Danmu_Main_mq.Push_tag(`c.Rev_add`,score)

			if rank_list := p.Json().GetArrayFrom(p.Json().GetValFromS(res, "data.OnlineRankItem"), "userRank");rank_list == nil {
				apilog.L(`E: `,"rank_list", len(rank_list))
				return
			} else if len(rank_list) == 0 {
				// apilog.L(`E: `,"rank_list == tmp_onlineNum")
				return
			} else {
				p.Sys().Timeoutf(1)
				self_loop(page+1)
				return
			}
		}
	}

	self_loop(1)
	apilog.Base("获取score").L(`W: `,"以往营收获取成功", fmt.Sprintf("%.2f", c.Rev))
	// c.Danmu_Main_mq.Push(c.Danmu_Main_mq_item{//传入消息队列
	// 	Class:`c.Rev_add`,
	// 	Data:self_loop(1),
	// })
	return
}

//获取热门榜
func (i *api) Get_HotRank() {
	if i.Uid == 0 || c.Roomid == 0 {
		apilog.Base_add("Get_HotRank").L(`E: `,"i.Uid == 0 || c.Roomid == 0")
		return
	}
	if api_limit.TO() {return}//超额请求阻塞，超时将取消
	apilog := apilog.Base_add(`获取热门榜`)

	Cookie := make(map[string]string)
	c.Cookie.Range(func(k,v interface{})(bool){
		Cookie[k.(string)] = v.(string)
		return true
	})

	req := p.Req()
	if err := req.Reqf(p.Rval{
		Url:`https://api.live.bilibili.com/xlive/general-interface/v1/rank/getHotRank?ruid=`+strconv.Itoa(i.Uid)+`&room_id=`+strconv.Itoa(c.Roomid)+`&is_pre=0&page_size=50&source=2&area_id=`+strconv.Itoa(i.Parent_area_id),
		Header:map[string]string{
			`Host`: `api.live.bilibili.com`,
			`User-Agent`: `Mozilla/5.0 (X11; Linux x86_64; rv:83.0) Gecko/20100101 Firefox/83.0`,
			`Accept`: `application/json, text/plain, */*`,
			`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
			`Accept-Encoding`: `gzip, deflate, br`,
			`Origin`: `https://live.bilibili.com`,
			`Connection`: `keep-alive`,
			`Pragma`: `no-cache`,
			`Cache-Control`: `no-cache`,
			`Referer`:"https://live.bilibili.com/" + strconv.Itoa(c.Roomid),
			`Cookie`:p.Map_2_Cookies_String(Cookie),
		},
		Timeout:3,
		Retry:2,
	});err != nil {
		apilog.L(`E: `,err)
		return
	}
	
	var type_item struct{
		Code int `json:"code"`
		Message string `json:"message"`
		Data struct{
			Own struct{
				Rank int `json:"rank"`
				Area_parent_name string `json:"area_parent_name"`
			} `json:"own"`
		} `json:"data"`
	}
	if e := json.Unmarshal(req.Respon, &type_item);e != nil {
		apilog.L(`E: `, e)
	}
	if type_item.Code != 0 {
		apilog.L(`E: `,type_item.Message)
		return
	}
	c.Note = type_item.Data.Own.Area_parent_name + " "
	if type_item.Data.Own.Rank == 0 {
		c.Note += `50+`
	} else {
		c.Note += strconv.Itoa(type_item.Data.Own.Rank)
	}
	apilog.L(`I: `,`热门榜:`,c.Note)
}

func (i *api) Get_guardNum() {
	if i.Uid == 0 || c.Roomid == 0 {
		apilog.Base_add("Get_guardNum").L(`E: `,"i.Uid == 0 || c.Roomid == 0")
		return
	}
	if api_limit.TO() {return}//超额请求阻塞，超时将取消
	apilog := apilog.Base_add(`获取舰长数`)

	Cookie := make(map[string]string)
	c.Cookie.Range(func(k,v interface{})(bool){
		Cookie[k.(string)] = v.(string)
		return true
	})

	req := p.Req()
	if err := req.Reqf(p.Rval{
		Url:`https://api.live.bilibili.com/xlive/app-room/v2/guardTab/topList?roomid=`+strconv.Itoa(c.Roomid)+`&page=1&ruid=`+strconv.Itoa(i.Uid)+`&page_size=29`,
		Header:map[string]string{
			`Host`: `api.live.bilibili.com`,
			`User-Agent`: `Mozilla/5.0 (X11; Linux x86_64; rv:83.0) Gecko/20100101 Firefox/83.0`,
			`Accept`: `application/json, text/plain, */*`,
			`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
			`Accept-Encoding`: `gzip, deflate, br`,
			`Origin`: `https://live.bilibili.com`,
			`Connection`: `keep-alive`,
			`Pragma`: `no-cache`,
			`Cache-Control`: `no-cache`,
			`Referer`:"https://live.bilibili.com/" + strconv.Itoa(c.Roomid),
			`Cookie`:p.Map_2_Cookies_String(Cookie),
		},
		Timeout:3,
		Retry:2,
	});err != nil {
		apilog.L(`E: `,err)
		return
	}
	res := string(req.Respon)
	if msg := p.Json().GetValFromS(res, "message");msg == nil || msg != "0" {
		apilog.L(`E: `,"message", msg)
		return
	}
	if num := p.Json().GetValFromS(res, "data.info.num");num == nil {
		apilog.L(`E: `,"num", num)
		return
	} else {
		c.GuardNum = int(num.(float64))
		apilog.L(`I: `,"当前舰长数", c.GuardNum)
	}
	return
}

func (i *api) Get_Version() {
	Roomid := strconv.Itoa(i.Roomid)
	if api_limit.TO() {return}//超额请求阻塞，超时将取消
	apilog := apilog.Base_add(`获取客户版本`)

	var player_js_url string
	{//获取player_js_url
		r := g.Get(p.Rval{
			Url:"https://live.bilibili.com/blanc/" + Roomid,
		})

		if r.Err != nil {
			apilog.L(`E: `,r.Err)
			return
		}

		r.S2(`<script src=`,`.js`)
		if r.Err != nil {
			apilog.L(`E: `,r.Err)
			return
		}

		for _,v := range r.RS {
			tmp := string(v) + `.js`
			if strings.Contains(tmp,`http`) {continue}
			tmp = `https:` + tmp
			if strings.Contains(tmp,`player`) {
				player_js_url = tmp
				break
			}
		}
		if player_js_url == `` {
			apilog.L(`E: `,`no found player js`)
			return
		}
	}

	{//获取VERSION
		r := g.Get(p.Rval{
			Url:player_js_url,
		})

		if r.Err != nil {
			apilog.L(`E: `,r.Err)
			return
		}

		r.S(`Bilibili HTML5 Live Player v`,` `,0,0)
		if r.Err != nil {
			apilog.L(`E: `,r.Err)
			return
		}
		c.VERSION = r.RS[0]
		apilog.L(`T: `,"api version", c.VERSION)
	}
}

//调用记录
var boot_Get_cookie funcCtrl.FlashFunc//新的替代旧的

//扫码登录
func Get_cookie() {
	if api_limit.TO() {return}//超额请求阻塞，超时将取消
	apilog := apilog.Base_add(`获取Cookie`)
	
	//获取id
	id := boot_Get_cookie.Flash()
	defer boot_Get_cookie.UnFlash()

	var img_url string
	var oauth string
	{//获取二维码
		r := p.Req()
		if e := r.Reqf(p.Rval{
			Url:`https://passport.bilibili.com/qrcode/getLoginUrl`,
			Timeout:10,
			Retry:2,
		});e != nil {
			apilog.L(`E: `,e)
			return
		}
		var res struct{
			Code int `json:"code"`
			Status bool `json:"status"`
			Data struct{
				Url string `json:"url"`
				OauthKey string `json:"oauthKey"`
			} `json:"data"`
		}
		if e := json.Unmarshal(r.Respon, &res);e != nil {
			apilog.L(`E: `, e)
			return
		}
		if res.Code != 0 {
			apilog.L(`E: `, `code != 0`)
			return
		}
		if !res.Status {
			apilog.L(`E: `, `status == false`)
			return
		}
		
		if res.Data.Url == `` {
			apilog.L(`E: `, `Data.Urls == ""`)
			return
		} else {img_url = res.Data.Url}
		if res.Data.OauthKey == `` {
			apilog.L(`E: `, `Data.OauthKey == ""`)
			return
		} else {oauth = res.Data.OauthKey}
	}

	//有新实例，退出
	if boot_Get_cookie.NeedExit(id) {return}

	var server = new(http.Server)
	{//生成二维码
		qr.WriteFile(img_url,qr.Medium,256,`qr.png`)
		if !p.Checkfile().IsExist(`qr.png`) {
			apilog.L(`E: `,`qr error`)
			return
		}
		//启动web
		s := web.New(server)
		s.Handle(map[string]func(http.ResponseWriter,*http.Request){
			`/`:func(w http.ResponseWriter,r *http.Request){
				var path string = r.URL.Path[1:]
				if path == `` {path = `index.html`}
				http.ServeFile(w, r, path)
			},
			`/exit`:func(w http.ResponseWriter,r *http.Request){
				s.Server.Shutdown(context.Background())
			},
		})
		defer server.Shutdown(context.Background())

		if c.K_v.LoadV(`扫码登录自动打开标签页`).(bool) {open.Run(`http://`+server.Addr+`/qr.png`)}
		apilog.L(`W: `,`打开链接扫码登录：`,`http://`+server.Addr+`/qr.png`)
		p.Sys().Timeoutf(1)
	}
	
	//有新实例，退出
	if boot_Get_cookie.NeedExit(id) {return}

	var cookie string
	{//3s刷新查看是否通过
		max_try := 20

		Cookie := make(map[string]string)
		c.Cookie.Range(func(k,v interface{})(bool){
			Cookie[k.(string)] = v.(string)
			return true
		})

		for max_try > 0 {
			max_try -= 1
			p.Sys().Timeoutf(3)
			
			//有新实例，退出
			if boot_Get_cookie.NeedExit(id) {return}

			r := p.Req()
			if e := r.Reqf(p.Rval{
				Url:`https://passport.bilibili.com/qrcode/getLoginInfo`,
				PostStr:`oauthKey=`+oauth,
				Header:map[string]string{
					`Content-Type`:`application/x-www-form-urlencoded; charset=UTF-8`,
					`Referer`: `https://passport.bilibili.com/login`,
					`Cookie`:p.Map_2_Cookies_String(Cookie),
				},
				Timeout:10,
				Retry:2,	
			});e != nil {
				apilog.L(`E: `,e)
				return
			}
			res := string(r.Respon)
			if v,ok := p.Json().GetValFromS(res, "status").(bool);!ok {
				apilog.L(`E: `,`getLoginInfo status false`)
				return
			} else if !v {
				if v,ok := p.Json().GetValFromS(res, "message").(string);ok {
					if max_try < 5 || max_try%5 == 0 {//减少日志频度
						apilog.L(`W: `,`登录中`,v,max_try)
					}
				}
				continue
			} else {
				apilog.L(`W: `,`登录，并保存了cookie`)
				if v := r.Response.Cookies();len(v) == 0 {
					apilog.L(`E: `,`getLoginInfo cookies len == 0`)
					return
				} else {
					cookie = p.Map_2_Cookies_String(p.Cookies_List_2_Map(v))//cookie to string
				}
				if cookie == `` {
					apilog.L(`E: `,`getLoginInfo cookies ""`)
					return
				} else {break}
			}
		}
		if max_try <= 0 {
			apilog.L(`W: `,`登录取消`)
			return
		}
		if len(cookie) == 0 {return}
	}

	//有新实例，退出
	if boot_Get_cookie.NeedExit(id) {return}

	{//写入cookie.txt
		for k,v := range p.Cookies_String_2_Map(cookie){
			c.Cookie.Store(k, v)
		}
		//生成cookieString
		cookieString := ``
		{
			c.Cookie.Range(func(k,v interface{})(bool){
				cookieString += k.(string)+`=`+v.(string)+`; `
				return true
			})
			t := []rune(cookieString)
			cookieString = string(t[:len(t)-2])
		}

		CookieSet([]byte(cookieString))
	}

	//有新实例，退出
	if boot_Get_cookie.NeedExit(id) {return}

	{//清理
		if p.Checkfile().IsExist(`qr.png`) {
			os.RemoveAll(`qr.png`)
			return
		}
	}
}

//短信登录
func Get_cookie_by_msg() {
	/*

	https://passport.bilibili.com/x/passport-login/web/sms/send


	*/
}

//牌子
type TGet_list_in_room struct{
	Medal_id int `json:"medal_id"`//牌子id
	Medal_name string `json:"medal_name"`//牌子名
	Target_id int `json:"target_id"`//牌子up主uid
	Target_name string `json:"target_name"`//牌子up主名
	Room_id int `json:"roomid"`//牌子直播间
	Last_wear_time int `json:"last_wear_time"`//佩戴有效截止时间（佩戴本身不会刷新，发弹幕，送小心心，送金瓜子礼物才会刷新）
	Today_intimacy int `json:"today_intimacy"`//今日亲密(0:未发送弹幕 100:已发送弹幕)
	Is_lighted int `json:"is_lighted"`//牌子是否熄灭(0:熄灭 1:亮)
}
//获取牌子信息
func Get_list_in_room() (array []TGet_list_in_room) {
	if api_limit.TO() {return}//超额请求阻塞，超时将取消
	apilog := apilog.Base_add(`获取牌子`)
	//验证cookie
	if missKey := CookieCheck([]string{
		`bili_jct`,
		`DedeUserID`,
		`LIVE_BUVID`,
	});len(missKey) != 0 {
		apilog.L(`T: `,`Cookie无Key:`,missKey)
		return
	}
	Cookie := make(map[string]string)
	c.Cookie.Range(func(k,v interface{})(bool){
		Cookie[k.(string)] = v.(string)
		return true
	})

	{//获取牌子列表
		var medalList []TGet_list_in_room
		for pageNum:=1; true;pageNum+=1{
			r := p.Req()
			if e := r.Reqf(p.Rval{
				Url:`https://api.live.bilibili.com/fans_medal/v5/live_fans_medal/iApiMedal?page=`+strconv.Itoa(pageNum)+`&pageSize=10`,
				Header:map[string]string{
					`Cookie`:p.Map_2_Cookies_String(Cookie),
				},
				Timeout:10,
				Retry:2,
			});e != nil {
				apilog.L(`E: `,e)
				return
			}
			
			var res struct{
				Code int `json:"code"`
				Msg string `json:"msg"`
				Message string `json:"message"`
				Data struct{
					FansMedalList []TGet_list_in_room `json"fansMedalList"`
					Pageinfo struct{
						Totalpages int `json:"totalpages"`
						CurPage int `json:"curPage"`
					} `json:"pageinfo"`
				} `json:"data"`
			}
	
			if e := json.Unmarshal(r.Respon, &res);e != nil{
				apilog.L(`E: `,e)
			}
	
			if res.Code != 0 {
				apilog.L(`E: `,`返回code`, res.Code, res.Msg)
				return
			}

			medalList = append(medalList, res.Data.FansMedalList...)

			if res.Data.Pageinfo.CurPage == res.Data.Pageinfo.Totalpages {break}

			time.Sleep(time.Second)
		}
		

		return medalList
	}
}

type TGet_weared_medal struct{
	Medal_id int `json:"medal_id"`//牌子id
	Medal_name string `json:"medal_name"`//牌子名
	Target_id int `json:"target_id"`//牌子up主uid
	Target_name string `json:"target_name"`//牌子up主名
	Roominfo Roominfo `json:"roominfo"`//牌子直播间
	Today_intimacy int `json:"today_intimacy"`//今日亲密(0:未发送弹幕 100:已发送弹幕)
	Is_lighted int `json:"is_lighted"`//牌子是否熄灭(0:熄灭 1:亮)
}
type Roominfo struct{
	Room_id int `json:"room_id"`
}
//获取当前佩戴的牌子
func Get_weared_medal() (item TGet_weared_medal) {
	if api_limit.TO() {return}//超额请求阻塞，超时将取消
	apilog := apilog.Base_add(`获取牌子`)
	//验证cookie
	if missKey := CookieCheck([]string{
		`bili_jct`,
		`DedeUserID`,
		`LIVE_BUVID`,
	});len(missKey) != 0 {
		apilog.L(`T: `,`Cookie无Key:`,missKey)
		return
	}
	Cookie := make(map[string]string)
	c.Cookie.Range(func(k,v interface{})(bool){
		Cookie[k.(string)] = v.(string)
		return true
	})

	{//获取
		r := p.Req()
		if e := r.Reqf(p.Rval{
			Url:`https://api.live.bilibili.com/live_user/v1/UserInfo/get_weared_medal`,
			Header:map[string]string{
				`Cookie`:p.Map_2_Cookies_String(Cookie),
			},
			Timeout:10,
			Retry:2,
		});e != nil {
			apilog.L(`E: `,e)
			return
		}

		var res struct{
			Code int `json:"code"`
			Msg	string `json:"msg"`
			Message	string `json:"message"`
			Data TGet_weared_medal `json:"data"`
		}
		if e := json.Unmarshal(r.Respon, &res);e != nil && res.Msg == ``{//未佩戴时的data是array型会导致错误
			apilog.L(`E: `,e)
			return
		}

		if res.Code != 0 {
			apilog.L(`E: `,`返回code`, res.Code, res.Msg)
			return
		}

		return res.Data
	}
	
}

func (i *api) CheckSwitch_FansMedal() {
	if api_limit.TO() {return}//超额请求阻塞，超时将取消
	apilog := apilog.Base_add(`切换粉丝牌`)
	//验证cookie
	if missKey := CookieCheck([]string{
		`bili_jct`,
		`DedeUserID`,
		`LIVE_BUVID`,
	});len(missKey) != 0 {
		apilog.L(`T: `,`Cookie无Key:`,missKey)
		return
	}

	Cookie := make(map[string]string)
	c.Cookie.Range(func(k,v interface{})(bool){
		Cookie[k.(string)] = v.(string)
		return true
	})
	{//获取当前牌子，验证是否本直播间牌子
		res := Get_weared_medal()

		c.Wearing_FansMedal = res.Roominfo.Room_id//更新佩戴信息
		if res.Target_id == c.UpUid {return}
	}

	var medal_id int//将要使用的牌子id
	//检查是否有此直播间的牌子
	{
		medal_list := Get_list_in_room()
		for _,v := range medal_list {
			if v.Target_id != c.UpUid {continue}
			medal_id = v.Medal_id
		}
		if medal_id == 0 {//无牌
			if c.Wearing_FansMedal == 0 {//当前没牌
				apilog.L(`I: `,`当前无粉丝牌，不切换`)
				return
			}
		}
	}

	var (
		post_url string
		post_str string
	)
	{//生成佩戴信息
		csrf,_ := c.Cookie.LoadV(`bili_jct`).(string)
		if csrf == `` {apilog.L(`E: `,"Cookie错误,无bili_jct=");return}
		
		post_str = `csrf_token=`+csrf+`&csrf=`+csrf
		
		if medal_id == 0 {//无牌，不佩戴牌子
			post_url = `https://api.live.bilibili.com/xlive/web-room/v1/fansMedal/take_off`
		} else {
			post_url = `https://api.live.bilibili.com/xlive/web-room/v1/fansMedal/wear`
			post_str = `medal_id=`+strconv.Itoa(medal_id)+`&`+post_str
		}
	}
	{//切换牌子
		r := p.Req()
		if e := r.Reqf(p.Rval{
			Url:post_url,
			PostStr:post_str,
			Header:map[string]string{
				`Cookie`:p.Map_2_Cookies_String(Cookie),
				`Content-Type`:`application/x-www-form-urlencoded; charset=UTF-8`,
				`Referer`: `https://passport.bilibili.com/login`,
			},
			Timeout:10,
			Retry:2,
		});e != nil {
			apilog.L(`E: `,e)
			return
		}
		res := string(r.Respon)
		if v,ok := p.Json().GetValFromS(res, "code").(float64);ok && v == 0 {
			apilog.L(`I: `,`自动切换粉丝牌`)
			c.Wearing_FansMedal = medal_id//更新佩戴信息
			return
		}
		if v,ok := p.Json().GetValFromS(res, "message").(string);ok {
			apilog.L(`E: `,`Get_FansMedal wear message`, v)
		} else {
			apilog.L(`E: `,`Get_FansMedal wear message nil`)
		}
	}
}

//签到
func Dosign() {
	apilog := apilog.Base_add(`签到`).L(`T: `,`签到`)
	//验证cookie
	if missKey := CookieCheck([]string{
		`bili_jct`,
		`DedeUserID`,
		`LIVE_BUVID`,
	});len(missKey) != 0 {
		apilog.L(`T: `,`Cookie无Key:`,missKey)
		return
	}
	if api_limit.TO() {return}//超额请求阻塞，超时将取消

	{//检查是否签到
		Cookie := make(map[string]string)
		c.Cookie.Range(func(k,v interface{})(bool){
			Cookie[k.(string)] = v.(string)
			return true
		})

		req := p.Req()
		if err := req.Reqf(p.Rval{
			Url:`https://api.live.bilibili.com/xlive/web-ucenter/v1/sign/WebGetSignInfo`,
			Header:map[string]string{
				`Host`: `api.live.bilibili.com`,
				`User-Agent`: `Mozilla/5.0 (X11; Linux x86_64; rv:83.0) Gecko/20100101 Firefox/83.0`,
				`Accept`: `application/json, text/plain, */*`,
				`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
				`Accept-Encoding`: `gzip, deflate, br`,
				`Origin`: `https://live.bilibili.com`,
				`Connection`: `keep-alive`,
				`Pragma`: `no-cache`,
				`Cache-Control`: `no-cache`,
				`Referer`:"https://live.bilibili.com/all",
				`Cookie`:p.Map_2_Cookies_String(Cookie),
			},
			Timeout:3,
			Retry:2,
		});err != nil {
			apilog.L(`E: `,err)
			return
		}
	
		var msg struct {
			Code int `json:"code"`
			Message string `json:"message"`
			Data struct {
				Status int `json:"status"`
			} `json:"data"`
		}
		if e := json.Unmarshal(req.Respon,&msg);e != nil{
			apilog.L(`E: `,e)
		}
		if msg.Code != 0 {apilog.L(`E: `,msg.Message);return}
		if msg.Data.Status == 1 {//今日已签到
			return
		}
	}

	{//签到
		Cookie := make(map[string]string)
		c.Cookie.Range(func(k,v interface{})(bool){
			Cookie[k.(string)] = v.(string)
			return true
		})

		req := p.Req()
		if err := req.Reqf(p.Rval{
			Url:`https://api.live.bilibili.com/xlive/web-ucenter/v1/sign/DoSign`,
			Header:map[string]string{
				`Host`: `api.live.bilibili.com`,
				`User-Agent`: `Mozilla/5.0 (X11; Linux x86_64; rv:83.0) Gecko/20100101 Firefox/83.0`,
				`Accept`: `application/json, text/plain, */*`,
				`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
				`Accept-Encoding`: `gzip, deflate, br`,
				`Origin`: `https://live.bilibili.com`,
				`Connection`: `keep-alive`,
				`Pragma`: `no-cache`,
				`Cache-Control`: `no-cache`,
				`Referer`:"https://live.bilibili.com/all",
				`Cookie`:p.Map_2_Cookies_String(Cookie),
			},
			Timeout:3,
			Retry:2,
		});err != nil {
			apilog.L(`E: `,err)
			return
		}
	
		var msg struct {
			Code int `json:"code"`
			Message string `json:"message"`
			Data struct {
				HadSignDays int `json:"hadSignDays"`
			} `json:"data"`
		}
		if e := json.Unmarshal(req.Respon,&msg);e != nil{
			apilog.L(`E: `,e)
		}
		if msg.Code == 0 {apilog.L(`I: `,`签到成功!本月已签到`, msg.Data.HadSignDays,`天`);return}
		apilog.L(`E: `,msg.Message)
	}
}

//LIVE_BUVID
func (i *api) Get_LIVE_BUVID() (o *api){
	o = i
	apilog := apilog.Base_add(`LIVE_BUVID`).L(`T: `,`获取`)
	if live_buvid,ok := c.Cookie.LoadV(`LIVE_BUVID`).(string);ok && live_buvid != `` {apilog.L(`T: `,`存在`);return}
	if c.Roomid == 0 {apilog.L(`E: `,`失败！无Roomid`);return}
	if api_limit.TO() {apilog.L(`E: `,`超时！`);return}//超额请求阻塞，超时将取消

	//当房间处于特殊活动状态时，将会获取不到，此处使用了若干著名up主房间进行尝试
	roomIdList := []string{
		strconv.Itoa(c.Roomid),//当前
		"3",//哔哩哔哩音悦台
		"2",//直播姬
		"1",//哔哩哔哩直播
	}

	for _,roomid := range roomIdList{//获取
		req := p.Req()
		if err := req.Reqf(p.Rval{
			Url:`https://api.live.bilibili.com/live/getRoomKanBanModel?roomid=`+roomid,
			Header:map[string]string{
				`Host`: `live.bilibili.com`,
				`User-Agent`: `Mozilla/5.0 (X11; Linux x86_64; rv:83.0) Gecko/20100101 Firefox/83.0`,
				`Accept`: `text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8`,
				`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
				`Accept-Encoding`: `gzip, deflate, br`,
				`Connection`: `keep-alive`,
				`Cache-Control`: `no-cache`,
				`Referer`:"https://live.bilibili.com",
				`DNT`: `1`,
				`Upgrade-Insecure-Requests`: `1`,
			},
			Timeout:3,
			Retry:2,
		});err != nil {
			apilog.L(`E: `,err)
			return
		}

		//cookie
		var has bool
		for k,v := range p.Cookies_List_2_Map(req.Response.Cookies()){
			c.Cookie.Store(k, v)
			if k == `LIVE_BUVID` {has = true}
		}
		if has {
			apilog.L(`I: `,`获取到LIVE_BUVID，保存cookie`)
			break
		} else {
			apilog.L(`I: `, roomid,`未获取到，重试`)
			time.Sleep(time.Second)
		}
	}
	
	Cookie := make(map[string]string)
	c.Cookie.Range(func(k,v interface{})(bool){
		Cookie[k.(string)] = v.(string)
		return true
	})
	
	CookieSet([]byte(p.Map_2_Cookies_String(Cookie)))

	return
}

//小心心
type E_json struct{
	Code int `json:"code"`
	Message string `json:"message"`
	Ttl int `json:"ttl"`
	Data struct{
		Timestamp int `json:"timestamp"`
		Heartbeat_interval int `json:"heartbeat_interval"`
		Secret_key string `json:"secret_key"`
		Secret_rule []int `json:"secret_rule"`
		Patch_status int `json:"patch_status"`
	} `json:"data"`
}

//调用记录
var boot_F_x25Kn funcCtrl.FlashFunc//新的替代旧的

func (i *api) F_x25Kn() (o *api) {
	o = i
	apilog := apilog.Base_add(`小心心`)
	if c.Wearing_FansMedal == 0{apilog.L(`I: `,`无粉丝牌，不获取`);return}
	//验证cookie
	if missKey := CookieCheck([]string{
		`bili_jct`,
		`DedeUserID`,
		`LIVE_BUVID`,
	});len(missKey) != 0 {
		apilog.L(`T: `,`Cookie无Key:`,missKey)
		return
	}
	if o.Parent_area_id == -1 {apilog.L(`E: `,`失败！未获取Parent_area_id`);return}
	if o.Area_id == -1 {apilog.L(`E: `,`失败！未获取Area_id`);return}
	if api_limit.TO() {apilog.L(`E: `,`超时！`);return}//超额请求阻塞，超时将取消

	id := boot_F_x25Kn.Flash()//获取函数调用会话id
	defer boot_F_x25Kn.UnFlash()

	{//查看今天小心心数量
		var num = 0
		for _,v := range Gift_list() {
			if v.Gift_id == 30607 && v.Expire_at - int(p.Sys().GetSTime()) > 6 * 86400 {
				num = v.Gift_num
			}
		}
		if num == 24 {
			Close(0)//关闭全部（0）浏览器websocket连接
			apilog.L(`I: `,`今天小心心已满！`);return
		} else {
			apilog.L(`I: `,`今天已有`,num,`个小心心，开始获取`)
			defer apilog.L(`T: `,`退出`)
		}
	}
	
	var (
		res E_json
		loop_num = 0
	)

	csrf,_ := c.Cookie.LoadV(`bili_jct`).(string)
	if csrf == `` {apilog.L(`E: `,"Cookie错误,无bili_jct");return}

	LIVE_BUVID := c.Cookie.LoadV(`LIVE_BUVID`).(string)
	if LIVE_BUVID == `` {apilog.L(`E: `,"Cookie错误,无LIVE_BUVID");return}

	var new_uuid string
	{
		if tmp_uuid,e := uuid.NewV4();e == nil {
			new_uuid = tmp_uuid.String()
		} else {
			apilog.L(`E: `,e)
			return
		}
	}

	{//初始化
		//新调用，此退出
		if boot_F_x25Kn.NeedExit(id) {return}

		PostStr := `id=[`+strconv.Itoa(o.Parent_area_id)+`,`+strconv.Itoa(o.Area_id)+`,`+strconv.Itoa(loop_num)+`,`+strconv.Itoa(o.Roomid)+`]&`
		PostStr += `device=["`+LIVE_BUVID+`","`+new_uuid+`"]&`
		PostStr += `ts=`+strconv.Itoa(int(p.Sys().GetMTime()))
		PostStr += `&is_patch=0&`
		PostStr += `heart_beat=[]&`
		PostStr += `ua=Mozilla/5.0 (X11; Linux x86_64; rv:84.0) Gecko/20100101 Firefox/84.0&`
		PostStr += `csrf_token=`+csrf+`&csrf=`+csrf+`&`
		PostStr += `visit_id=`

		Cookie := make(map[string]string)
		c.Cookie.Range(func(k,v interface{})(bool){
			Cookie[k.(string)] = v.(string)
			return true
		})

		req := p.Req()
		if err := req.Reqf(p.Rval{
			Url:`https://live-trace.bilibili.com/xlive/data-interface/v1/x25Kn/E`,
			Header:map[string]string{
				`Host`: `api.live.bilibili.com`,
				`User-Agent`: `Mozilla/5.0 (X11; Linux x86_64; rv:83.0) Gecko/20100101 Firefox/83.0`,
				`Accept`: `application/json, text/plain, */*`,
				`Content-Type`: `application/x-www-form-urlencoded`,
				`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
				`Accept-Encoding`: `gzip, deflate, br`,
				`Origin`: `https://live.bilibili.com`,
				`Connection`: `keep-alive`,
				`Pragma`: `no-cache`,
				`Cache-Control`: `no-cache`,
				`Referer`:"https://live.bilibili.com/"+strconv.Itoa(o.Roomid),
				`Cookie`:p.Map_2_Cookies_String(Cookie),
			},
			PostStr:url.PathEscape(PostStr),
			Timeout:3,
			Retry:2,
		});err != nil {
			apilog.L(`E: `,err)
			return
		}

		if e := json.Unmarshal(req.Respon,&res);e != nil {
			apilog.L(`E: `,e)
			return
		}

		if res.Code != 0{
			apilog.L(`E: `,`返回错误`,res.Message)
			return
		}
	}

	{//loop
		for loop_num < (24+2)*5 {
			loop_num += 1
			//查看今天小心心数量
			if loop_num%5 == 0 {//每5min
				{//查看今天小心心数量
					var num = 0
					for _,v := range Gift_list() {
						if v.Gift_id == 30607 && v.Expire_at - int(p.Sys().GetSTime()) > 6 * 86400 {
							num = v.Gift_num
						}
					}
					if num == 24 {
						Close(0)//关闭全部（0）浏览器websocket连接
						apilog.L(`I: `,`今天小心心已满！`);return
					} else {
						apilog.L(`I: `,`获取了今天的第`,num,`个小心心`)
					}
				}
			}

			<- time.After(time.Second*time.Duration(res.Data.Heartbeat_interval))
			
			//新调用，此退出
			if boot_F_x25Kn.NeedExit(id) {return}

			var rt_obj = RT{
				R:R{
					Id:`[`+strconv.Itoa(o.Parent_area_id)+`,`+strconv.Itoa(o.Area_id)+`,`+strconv.Itoa(loop_num)+`,`+strconv.Itoa(o.Roomid)+`]`,
					Device:`["`+LIVE_BUVID+`","`+new_uuid+`"]`,
					Ets:res.Data.Timestamp,
					Benchmark:res.Data.Secret_key,
					Time:res.Data.Heartbeat_interval,
					Ts:int(p.Sys().GetMTime()),
				},
				T:res.Data.Secret_rule,
			}

			PostStr := `id=`+rt_obj.R.Id+`&`
			PostStr += `device=["`+LIVE_BUVID+`","`+new_uuid+`"]&`
			PostStr += `ets=`+strconv.Itoa(res.Data.Timestamp)
			PostStr += `&benchmark=`+res.Data.Secret_key
			PostStr += `&time=`+strconv.Itoa(res.Data.Heartbeat_interval)
			PostStr += `&ts=`+strconv.Itoa(rt_obj.R.Ts)
			PostStr += `&is_patch=0&`
			PostStr += `heart_beat=[]&`
			PostStr += `ua=Mozilla/5.0 (X11; Linux x86_64; rv:84.0) Gecko/20100101 Firefox/84.0&`
			PostStr += `csrf_token=`+csrf+`&csrf=`+csrf+`&`
			PostStr += `visit_id=`
			
			if wasm := Wasm(3, 0, rt_obj);wasm == `` {//0全局
				apilog.L(`E: `,`发生错误`)
				return
			} else {
				PostStr = `s=`+wasm+`&`+PostStr
			}

			Cookie := make(map[string]string)
			c.Cookie.Range(func(k,v interface{})(bool){
				Cookie[k.(string)] = v.(string)
				return true
			})

			req := p.Req()
			if err := req.Reqf(p.Rval{
				Url:`https://live-trace.bilibili.com/xlive/data-interface/v1/x25Kn/X`,
				Header:map[string]string{
					`Host`: `api.live.bilibili.com`,
					`User-Agent`: `Mozilla/5.0 (X11; Linux x86_64; rv:83.0) Gecko/20100101 Firefox/83.0`,
					`Accept`: `application/json, text/plain, */*`,
					`Content-Type`: `application/x-www-form-urlencoded`,
					`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
					`Accept-Encoding`: `gzip, deflate, br`,
					`Origin`: `https://live.bilibili.com`,
					`Connection`: `keep-alive`,
					`Pragma`: `no-cache`,
					`Cache-Control`: `no-cache`,
					`Referer`:"https://live.bilibili.com/"+strconv.Itoa(o.Roomid),
					`Cookie`:p.Map_2_Cookies_String(Cookie),
				},
				PostStr:url.PathEscape(PostStr),
				Timeout:3,
				Retry:2,
			});err != nil {
				apilog.L(`E: `,err)
				return
			}

			if e := json.Unmarshal(req.Respon,&res);e != nil {
				apilog.L(`E: `,e)
				return
			}
	
			if res.Code != 0{
				apilog.L(`E: `,`返回错误`,res.Message)
				return
			}
		}
	}
	return
}

//礼物列表
type Gift_list_type struct {
	Code int `json:"code"`
	Message string `json:"message"`
	Data Gift_list_type_Data `json:"data"`
}

type Gift_list_type_Data struct {
	List []Gift_list_type_Data_List `json:"list"`
}

type Gift_list_type_Data_List struct{
	Bag_id int `json:"bag_id"`
	Gift_id int `json:"gift_id"`
	Gift_name string `json:"gift_name"`
	Gift_num int `json:"gift_num"`
	Expire_at int `json:"expire_at"`
}

func Gift_list() (list []Gift_list_type_Data_List) {
	apilog := apilog.Base_add(`礼物列表`)
	//验证cookie
	if missKey := CookieCheck([]string{
		`bili_jct`,
		`DedeUserID`,
		`LIVE_BUVID`,
	});len(missKey) != 0 {
		apilog.L(`T: `,`Cookie无Key:`,missKey)
		return
	}
	if c.Roomid == 0 {apilog.L(`E: `,`失败！无Roomid`);return}
	if api_limit.TO() {apilog.L(`E: `,`超时！`);return}//超额请求阻塞，超时将取消

	Cookie := make(map[string]string)
	c.Cookie.Range(func(k,v interface{})(bool){
		Cookie[k.(string)] = v.(string)
		return true
	})

	req := p.Req()
	if err := req.Reqf(p.Rval{
		Url:`https://api.live.bilibili.com/xlive/web-room/v1/gift/bag_list?t=`+strconv.Itoa(int(p.Sys().GetMTime()))+`&room_id=`+strconv.Itoa(c.Roomid),
		Header:map[string]string{
			`Host`: `api.live.bilibili.com`,
			`User-Agent`: `Mozilla/5.0 (X11; Linux x86_64; rv:83.0) Gecko/20100101 Firefox/83.0`,
			`Accept`: `application/json, text/plain, */*`,
			`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
			`Accept-Encoding`: `gzip, deflate, br`,
			`Origin`: `https://live.bilibili.com`,
			`Connection`: `keep-alive`,
			`Pragma`: `no-cache`,
			`Cache-Control`: `no-cache`,
			`Referer`:"https://live.bilibili.com/"+strconv.Itoa(c.Roomid),
			`Cookie`:p.Map_2_Cookies_String(Cookie),
		},
		Timeout:3,
		Retry:2,
	});err != nil {
		apilog.L(`E: `,err)
		return
	}

	var res Gift_list_type

	if e := json.Unmarshal(req.Respon,&res);e != nil {
		apilog.L(`E: `,e)
		return
	}

	if res.Code != 0{
		apilog.L(`E: `,res.Message)
		return
	}

	apilog.L(`T: `,`成功`)
	return res.Data.List
}

//银瓜子2硬币
func Silver_2_coin() {
	apilog := apilog.Base_add(`银瓜子=>硬币`).L(`T: `,`开始`)
	//验证cookie
	if missKey := CookieCheck([]string{
		`bili_jct`,
		`DedeUserID`,
		`LIVE_BUVID`,
	});len(missKey) != 0 {
		apilog.L(`T: `,`Cookie无Key:`,missKey)
		return
	}
	if api_limit.TO() {apilog.L(`E: `,`超时！`);return}//超额请求阻塞，超时将取消

	var Silver int
	{//验证是否还有机会
		Cookie := make(map[string]string)
		c.Cookie.Range(func(k,v interface{})(bool){
			Cookie[k.(string)] = v.(string)
			return true
		})

		req := p.Req()
		if err := req.Reqf(p.Rval{
			Url:`https://api.live.bilibili.com/pay/v1/Exchange/getStatus`,
			Header:map[string]string{
				`Host`: `api.live.bilibili.com`,
				`User-Agent`: `Mozilla/5.0 (X11; Linux x86_64; rv:83.0) Gecko/20100101 Firefox/83.0`,
				`Accept`: `application/json, text/plain, */*`,
				`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
				`Accept-Encoding`: `gzip, deflate, br`,
				`Origin`: `https://link.bilibili.com`,
				`Connection`: `keep-alive`,
				`Pragma`: `no-cache`,
				`Cache-Control`: `no-cache`,
				`Referer`:`https://link.bilibili.com/p/center/index`,
				`Cookie`:p.Map_2_Cookies_String(Cookie),
			},
			Timeout:3,
			Retry:2,
		});err != nil {
			apilog.L(`E: `,err)
			return
		}
	
		var res struct{
			Code int `json:"code"`
			Msg string `json:"msg"`
			Message string `json:"message"`
			Data struct{
				Silver int `json:"silver"`
				Silver_2_coin_left int `json:"silver_2_coin_left"`
			} `json:"data"`
		}
	
		if e := json.Unmarshal(req.Respon, &res);e != nil{
			apilog.L(`E: `, e)
			return
		}
	
		if res.Code != 0{
			apilog.L(`E: `, res.Message)
			return
		}

		if res.Data.Silver_2_coin_left == 0{
			apilog.L(`I: `, `今天次数已用完`)
			return
		}

		apilog.L(`T: `, `现在有银瓜子`, res.Data.Silver, `个`)
		Silver = res.Data.Silver
	}

	{//获取交换规则，验证数量足够
		Cookie := make(map[string]string)
		c.Cookie.Range(func(k,v interface{})(bool){
			Cookie[k.(string)] = v.(string)
			return true
		})

		req := p.Req()
		if err := req.Reqf(p.Rval{
			Url:`https://api.live.bilibili.com/pay/v1/Exchange/getRule`,
			Header:map[string]string{
				`Host`: `api.live.bilibili.com`,
				`User-Agent`: `Mozilla/5.0 (X11; Linux x86_64; rv:83.0) Gecko/20100101 Firefox/83.0`,
				`Accept`: `application/json, text/plain, */*`,
				`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
				`Accept-Encoding`: `gzip, deflate, br`,
				`Origin`: `https://link.bilibili.com`,
				`Connection`: `keep-alive`,
				`Pragma`: `no-cache`,
				`Cache-Control`: `no-cache`,
				`Referer`:`https://link.bilibili.com/p/center/index`,
				`Cookie`:p.Map_2_Cookies_String(Cookie),
			},
			Timeout:3,
			Retry:2,
		});err != nil {
			apilog.L(`E: `,err)
			return
		}
	
		var res struct{
			Code int `json:"code"`
			Msg string `json:"msg"`
			Message string `json:"message"`
			Data struct{
				Silver_2_coin_price int `json:"silver_2_coin_price"`
			} `json:"data"`
		}
	
		if e := json.Unmarshal(req.Respon, &res);e != nil{
			apilog.L(`E: `, e)
			return
		}
	
		if res.Code != 0{
			apilog.L(`E: `, res.Message)
			return
		}

		if Silver < res.Data.Silver_2_coin_price{
			apilog.L(`W: `, `当前银瓜子数量不足`)
			return
		}
	}
	
	{//交换
		csrf,_ := c.Cookie.LoadV(`bili_jct`).(string)
		if csrf == `` {apilog.L(`E: `,"Cookie错误,无bili_jct=");return}
		
		post_str := `csrf_token=`+csrf+`&csrf=`+csrf

		Cookie := make(map[string]string)
		c.Cookie.Range(func(k,v interface{})(bool){
			Cookie[k.(string)] = v.(string)
			return true
		})

		req := p.Req()
		if err := req.Reqf(p.Rval{
			Url:`https://api.live.bilibili.com/pay/v1/Exchange/silver2coin`,
			PostStr:url.PathEscape(post_str),
			Header:map[string]string{
				`Host`: `api.live.bilibili.com`,
				`User-Agent`: `Mozilla/5.0 (X11; Linux x86_64; rv:83.0) Gecko/20100101 Firefox/83.0`,
				`Accept`: `application/json, text/plain, */*`,
				`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
				`Accept-Encoding`: `gzip, deflate, br`,
				`Origin`: `https://link.bilibili.com`,
				`Connection`: `keep-alive`,
				`Pragma`: `no-cache`,
				`Cache-Control`: `no-cache`,
				`Content-Type`: `application/x-www-form-urlencoded`,
				`Referer`:`https://link.bilibili.com/p/center/index`,
				`Cookie`:p.Map_2_Cookies_String(Cookie),
			},
			Timeout:3,
			Retry:2,
		});err != nil {
			apilog.L(`E: `,err)
			return
		}
	
		save_cookie(req.Response.Cookies())

		var res struct{
			Code int `json:"code"`
			Msg string `json:"msg"`
			Message string `json:"message"`
		}
	
		if e := json.Unmarshal(req.Respon, &res);e != nil{
			apilog.L(`E: `, e)
			return
		}
	
		if res.Code != 0{
			apilog.L(`E: `, res.Message)
			return
		}
		apilog.L(`I: `, res.Message)
	}
}

func save_cookie(Cookies []*http.Cookie){
	for k,v := range p.Cookies_List_2_Map(Cookies){
		c.Cookie.Store(k, v)
	}

	Cookie := make(map[string]string)
	c.Cookie.Range(func(k,v interface{})(bool){
		Cookie[k.(string)] = v.(string)
		return true
	})
	CookieSet([]byte(p.Map_2_Cookies_String(Cookie)))
}

//正在直播主播
type UpItem struct{
	Uname string `json:"uname"`
	Title string `json:"title"`
	Roomid int `json:"roomid"`
}
func Feed_list() (Uplist []UpItem) {
	apilog := apilog.Base_add(`正在直播主播`).L(`T: `,`获取中`)
	//验证cookie
	if missKey := CookieCheck([]string{
		`bili_jct`,
		`DedeUserID`,
		`LIVE_BUVID`,
	});len(missKey) != 0 {
		apilog.L(`T: `,`Cookie无Key:`,missKey)
		return
	}
	if api_limit.TO() {apilog.L(`E: `,`超时！`);return}//超额请求阻塞，超时将取消

	Cookie := make(map[string]string)
	c.Cookie.Range(func(k,v interface{})(bool){
		Cookie[k.(string)] = v.(string)
		return true
	})

	req := p.Req()
	for pageNum:=1; true; pageNum+=1 {
		if err := req.Reqf(p.Rval{
			Url:`https://api.live.bilibili.com/relation/v1/feed/feed_list?page=`+strconv.Itoa(pageNum)+`&pagesize=10`,
			Header:map[string]string{
				`Host`: `api.live.bilibili.com`,
				`User-Agent`: `Mozilla/5.0 (X11; Linux x86_64; rv:83.0) Gecko/20100101 Firefox/83.0`,
				`Accept`: `application/json, text/plain, */*`,
				`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
				`Accept-Encoding`: `gzip, deflate, br`,
				`Origin`: `https://t.bilibili.com`,
				`Connection`: `keep-alive`,
				`Pragma`: `no-cache`,
				`Cache-Control`: `no-cache`,
				`Referer`:`https://t.bilibili.com/pages/nav/index_new`,
				`Cookie`:p.Map_2_Cookies_String(Cookie),
			},
			Timeout:3,
			Retry:2,
		});err != nil {
			apilog.L(`E: `,err)
			return
		}

		var res struct{
			Code int `json:"code"`
			Msg string `json:"msg"`
			Message string `json:"message"`
			Data struct{
				Results int `json:"results"`
				List []UpItem `json:"list"`
			} `json:"data"`
		}

		if e := json.Unmarshal(req.Respon, &res);e != nil{
			apilog.L(`E: `, e)
			return
		}

		if res.Code != 0{
			apilog.L(`E: `, res.Message)
			return
		}

		Uplist = append(Uplist, res.Data.List...)

		if pageNum*10 > res.Data.Results {break}
		time.Sleep(time.Second)
	}

	apilog.L(`T: `,`完成`)
	return
}