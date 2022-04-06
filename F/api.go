package F

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	c "github.com/qydysky/bili_danmu/CV"
	J "github.com/qydysky/bili_danmu/Json"
	"github.com/skratchdot/open-golang/open"

	p "github.com/qydysky/part"
	funcCtrl "github.com/qydysky/part/funcCtrl"
	g "github.com/qydysky/part/get"
	limit "github.com/qydysky/part/limit"
	reqf "github.com/qydysky/part/reqf"
	sync "github.com/qydysky/part/sync"
	web "github.com/qydysky/part/web"

	uuid "github.com/gofrs/uuid"
	"github.com/mdp/qrterminal/v3"
	qr "github.com/skip2/go-qrcode"
)

var apilog = c.Log.Base(`api`)
var api_limit = limit.New(1, 2000, 30000) //频率限制1次/2s，最大等待时间30s

func Get(key string, kv sync.Map) {
	apilog := apilog.Base_add(`Get`)

	if api_limit.TO() {
		return
	} //超额请求阻塞，超时将取消

	var api_can_get = map[string][]func(sync.Map) []string{
		`Cookie`: { //Cookie
			Get_cookie,
		},
		`Uid`: { //用戶uid
			GetUid,
		},
		`UpUid`: { //主播uid
			getInfoByRoom,
			getRoomPlayInfo,
			Html,
		},
		`Live_Start_Time`: { //直播开始时间
			getInfoByRoom,
			getRoomPlayInfo,
			Html,
		},
		`Liveing`: { //是否在直播
			getInfoByRoom,
			getRoomPlayInfo,
			Html,
		},
		`Title`: { //直播间标题
			getInfoByRoom,
			Html,
		},
		`Uname`: { //主播名
			getInfoByRoom,
			Html,
		},
		`ParentAreaID`: { //分区
			getInfoByRoom,
			Html,
		},
		`AreaID`: { //子分区
			getInfoByRoom,
			Html,
		},
		`Roomid`: { //房间id
			missRoomId,
		},
		`GuardNum`: { //舰长数
			Get_guardNum,
			getInfoByRoom,
			getRoomPlayInfo,
			Html,
		},
		`Note`: { //分区排行
			Get_HotRank,
			getInfoByRoom,
			Html,
		},
		`Locked`: { //直播间是否被封禁
			getInfoByRoom,
			Html,
		},
		`Live_qn`: { //当前直播流质量
			getRoomPlayInfo,
			Html,
		},
		`AcceptQn`: { //允许的清晰度
			getRoomPlayInfo,
			Html,
		},
		`Live`: { //直播流链接
			getRoomPlayInfoByQn,
			getRoomPlayInfo,
			Html,
		},
		`Token`: { //弹幕钥
			getDanmuInfo,
		},
		`WSURL`: { //弹幕链接
			getDanmuInfo,
		},
		// `VERSION`:[]func()([]string){//客户版本  不再需要
		// 	Get_Version,
		// },
		`LIVE_BUVID`: { //LIVE_BUVID
			Get_LIVE_BUVID,
		},

		`Silver_2_coin`: { //银瓜子2硬币
			Silver_2_coin,
		},
		`CheckSwitch_FansMedal`: { //切换粉丝牌
			CheckSwitch_FansMedal,
		},
	}
	var check = map[string]func() bool{
		`Uid`: func() bool { //用戶uid
			return c.Uid != 0
		},
		`UpUid`: func() bool { //主播uid
			return c.UpUid != 0
		},
		`Live_Start_Time`: func() bool { //直播开始时间
			return c.Live_Start_Time != time.Time{}
		},
		`Liveing`: func() bool { //是否在直播
			return true
		},
		`Title`: func() bool { //直播间标题
			return c.Title != ``
		},
		`Uname`: func() bool { //主播名
			return c.Uname != ``
		},
		`ParentAreaID`: func() bool { //分区
			return c.ParentAreaID != 0
		},
		`AreaID`: func() bool { //子分区
			return c.AreaID != 0
		},
		`Roomid`: func() bool { //房间id
			return c.Roomid != 0
		},
		`GuardNum`: func() bool { //舰长数
			return c.GuardNum != 0
		},
		`Note`: func() bool { //分区排行
			return c.Note != ``
		},
		`Locked`: func() bool { //直播间是否被封禁
			return true
		},
		`Live_qn`: func() bool { //当前直播流质量
			return c.Live_qn != 0
		},
		`AcceptQn`: func() bool { //允许的清晰度
			return len(c.AcceptQn) != 0
		},
		`Live`: func() bool { //直播流链接
			return len(c.Live) != 0
		},
		`Token`: func() bool { //弹幕钥
			return c.Token != ``
		},
		`WSURL`: func() bool { //弹幕链接
			return len(c.WSURL) != 0
		},
		// `VERSION`:func()(bool){//客户版本  不再需要
		// 	return c.VERSION != `2.0.11`
		// },
		`LIVE_BUVID`: func() bool { //LIVE_BUVID
			return c.LIVE_BUVID
		},
		`Silver_2_coin`: func() bool { //银瓜子2硬币
			return true
		},
		`CheckSwitch_FansMedal`: func() bool { //切换粉丝牌
			return true
		},
		`Cookie`: func() bool { //Cookie
			return true
		},
	}

	if fList, ok := api_can_get[key]; ok {
		for _, fItem := range fList {
			apilog.L(`T: `, `Get`, key)
			missKey := fItem(kv)
			if len(missKey) > 0 {
				apilog.L(`T: `, `missKey when get`, key, missKey)
				for _, misskeyitem := range missKey {
					if checkf, ok := check[misskeyitem]; ok && checkf() {
						continue
					}
					if misskeyitem == key {
						apilog.L(`W: `, `missKey equrt key`, key, missKey)
						continue
					}
					Get(misskeyitem, kv)
				}
				missKey := fItem(kv)
				if len(missKey) > 0 {
					apilog.L(`W: `, `missKey when get`, key, missKey)
					continue
				}
			}
			if checkf, ok := check[key]; ok && checkf() {
				break
			}
		}
	}
}

func GetUid(kv sync.Map) (missKey []string) {
	if uid, ok := kv.LoadV(`Cookie`).(map[string]string)[`DedeUserID`]; !ok { //cookie中无DedeUserID
		missKey = append(missKey, `Cookie`)
	} else if uid, e := strconv.Atoi(uid); e != nil {
		missKey = append(missKey, `Cookie`)
	} else {
		c.Uid = uid
	}
	return
}

func Info(UpUid int) (info J.Info) {
	apilog := apilog.Base_add(`Info`)
	if api_limit.TO() {
		return
	} //超额请求阻塞，超时将取消

	//html
	{
		req := reqf.New()
		if err := req.Reqf(reqf.Rval{
			Url:     `https://api.bilibili.com/x/space/acc/info?mid=` + strconv.Itoa(UpUid) + `&jsonp=jsonp`,
			Proxy:   kv.LoadV(`Common.Proxy`).(string),
			Timeout: 10 * 1000,
			Retry:   2,
		}); err != nil {
			apilog.L(`E: `, err)
			return
		}

		//Info
		{
			if e := json.Unmarshal(req.Respon, &info); e != nil {
				apilog.L(`E: `, e)
				return
			} else if info.Code != 0 {
				apilog.L(`E: `, info.Message)
				return
			}
		}
	}
	return
}

func Html() (missKey []string) {
	apilog := apilog.Base_add(`html`)

	if c.Roomid == 0 {
		missKey = append(missKey, `Roomid`)
		return
	}

	Roomid := strconv.Itoa(c.Roomid)

	//html
	{
		r := g.Get(reqf.Rval{
			Url:   "https://live.bilibili.com/" + Roomid,
			Proxy: kv.LoadV(`Common.Proxy`).(string),
		})

		if tmp := r.S(`<script>window.__NEPTUNE_IS_MY_WAIFU__=`, `</script>`, 0, 0); tmp.Err != nil {
			apilog.L(`E: `, `不存在<script>window.__NEPTUNE_IS_MY_WAIFU__= </script>`)
		} else {
			s := tmp.RS[0]

			//Roominitres
			{
				var j struct {
					Roominitres J.Roominitres `json:"roomInitRes"`
				}
				if e := json.Unmarshal([]byte(s), &j); e != nil {
					apilog.L(`E: `, e)
					return
				} else if j.Roominitres.Code != 0 {
					apilog.L(`E: `, j.Roominitres.Message)
					return
				}

				//主播uid
				c.UpUid = j.Roominitres.Data.UID
				//房间号（完整）
				if j.Roominitres.Data.RoomID != 0 {
					c.Roomid = j.Roominitres.Data.RoomID
				}
				//直播开始时间
				c.Live_Start_Time = time.Unix(int64(j.Roominitres.Data.LiveTime), 0)
				//是否在直播
				c.Liveing = j.Roominitres.Data.LiveStatus == 1

				//未在直播，不获取直播流
				if !c.Liveing {
					c.Live_qn = 0
					c.AcceptQn = c.Qn
					c.Live = []string{}
					return
				}

				//当前直播流
				{
					type Stream_name struct {
						Protocol_name string
						Format_name   string
						Codec_name    string
					}
					var name_map = map[string]Stream_name{
						`flv`: {
							Protocol_name: "http_stream",
							Format_name:   "flv",
							Codec_name:    "avc",
						},
						`hls`: {
							Protocol_name: "http_hls",
							Format_name:   "fmp4",
							Codec_name:    "avc",
						},
					}

					want_type := name_map[`hls`]
					if v, ok := c.K_v.LoadV(`直播流类型`).(string); ok {
						if v, ok := name_map[v]; ok {
							want_type = v
						} else {
							apilog.L(`I: `, `未找到`, v, `,默认hls`)
						}
					} else {
						apilog.L(`T: `, `默认flv`)
					}

					for _, v := range j.Roominitres.Data.PlayurlInfo.Playurl.Stream {
						if v.ProtocolName != want_type.Protocol_name {
							continue
						}

						for _, v := range v.Format {
							if v.FormatName != want_type.Format_name {
								continue
							}

							for _, v := range v.Codec {
								if v.CodecName != want_type.Codec_name {
									continue
								}

								//当前直播流质量
								c.Live_qn = v.CurrentQn
								if c.Live_want_qn == 0 {
									c.Live_want_qn = v.CurrentQn
								}
								//允许的清晰度
								{
									var tmp = make(map[int]string)
									for _, v := range v.AcceptQn {
										if s, ok := c.Qn[v]; ok {
											tmp[v] = s
										}
									}
									c.AcceptQn = tmp
								}
								//直播流链接
								c.Live = []string{}
								for _, v1 := range v.URLInfo {
									c.Live = append(c.Live, v1.Host+v.BaseURL+v1.Extra)
								}
							}
						}
					}
				}
			}

			//Roominfores
			{
				var j struct {
					Roominfores J.Roominfores `json:"roomInitRes"`
				}

				if e := json.Unmarshal([]byte(s), &j); e != nil {
					apilog.L(`E: `, e)
					return
				} else if j.Roominfores.Code != 0 {
					apilog.L(`E: `, j.Roominfores.Message)
					return
				}

				//直播间标题
				c.Title = j.Roominfores.Data.RoomInfo.Title
				//主播名
				c.Uname = j.Roominfores.Data.AnchorInfo.BaseInfo.Uname
				//分区
				c.ParentAreaID = j.Roominfores.Data.RoomInfo.ParentAreaID
				//子分区
				c.AreaID = j.Roominfores.Data.RoomInfo.AreaID
				//舰长数
				c.GuardNum = j.Roominfores.Data.GuardInfo.Count
				//分区排行
				c.Note = j.Roominfores.Data.HotRankInfo.AreaName
				if rank := j.Roominfores.Data.HotRankInfo.Rank; rank > 50 || rank == 0 {
					c.Note += "50+"
				} else {
					c.Note += strconv.Itoa(rank)
				}
				//直播间是否被封禁
				if j.Roominfores.Data.RoomInfo.LockStatus == 1 {
					apilog.L(`W: `, "直播间封禁中")
					c.Locked = true
					return
				}
			}
		}
	}
	return
}

func missRoomId() (missKey []string) {
	apilog.Base_add(`missRoomId`).L(`E: `, `missRoomId`)
	return
}

func getInfoByRoom(kv sync.Map) (missKey []string) {
	apilog := apilog.Base_add(`getInfoByRoom`)

	var roomId = kv.LoadV(`common.Roomid`).(int)

	if roomId == 0 {
		missKey = append(missKey, `Roomid`)
		return
	}

	Roomid := strconv.Itoa(roomId)

	{ //使用其他api
		req := reqf.New()
		if err := req.Reqf(reqf.Rval{
			Url: "https://api.live.bilibili.com/xlive/web-room/v1/index/getInfoByRoom?room_id=" + Roomid,
			Header: map[string]string{
				`Referer`: "https://live.bilibili.com/" + Roomid,
			},
			Proxy:   kv.LoadV(`common.Proxy`).(string),
			Timeout: 10 * 1000,
			Retry:   2,
		}); err != nil {
			apilog.L(`E: `, err)
			return
		}

		//Roominfores
		{
			var j J.Roominfores

			if e := json.Unmarshal(req.Respon, &j); e != nil {
				apilog.L(`E: `, e)
				return
			} else if j.Code != 0 {
				apilog.L(`E: `, j.Message)
				return
			}

			//直播开始时间
			kv.Store(`common.Live_Start_Time`, time.Unix(int64(j.Data.RoomInfo.LiveStartTime), 0))
			//是否在直播
			kv.Store(`common.Liveing`, j.Data.RoomInfo.LiveStatus == 1)
			//直播间标题
			kv.Store(`common.Title`, j.Data.RoomInfo.Title)
			//主播名
			kv.Store(`common.Uname`, j.Data.AnchorInfo.BaseInfo.Uname)
			//分区
			kv.Store(`common.ParentAreaID`, j.Data.RoomInfo.ParentAreaID)
			//子分区
			kv.Store(`common.AreaID`, j.Data.RoomInfo.AreaID)
			//主播id
			kv.Store(`common.UpUid`, j.Data.RoomInfo.UID)
			//房间id
			if j.Data.RoomInfo.RoomID != 0 {
				kv.Store(`common.Roomid`, j.Data.RoomInfo.RoomID)
			}
			//舰长数
			kv.Store(`common.GuardNum`, j.Data.GuardInfo.Count)
			//分区排行
			{
				var note = j.Data.HotRankInfo.AreaName
				if rank := j.Data.HotRankInfo.Rank; rank > 50 || rank == 0 {
					note += "50+"
				} else {
					note += strconv.Itoa(rank)
				}
				kv.Store(`common.Note`, note)
			}
			//直播间是否被封禁
			if j.Data.RoomInfo.LockStatus == 1 {
				apilog.L(`W: `, "直播间封禁中")
				kv.Store(`common.Locked`, true)
				return
			}
		}
	}
	return
}

func getRoomPlayInfo(kv sync.Map) (missKey []string) {
	apilog := apilog.Base_add(`getRoomPlayInfo`)

	if kv.LoadV(`Roomid`).(int) == 0 {
		missKey = append(missKey, `Roomid`)
	}
	if !kv.LoadV(`LIVE_BUVID`).(bool) {
		missKey = append(missKey, `LIVE_BUVID`)
	}
	if len(missKey) > 0 {
		return
	}

	Roomid := strconv.Itoa(kv.LoadV(`Roomid`).(int))

	//Roominitres
	{
		Cookie := kv.LoadV(`Cookie`).(map[string]string)

		req := reqf.New()
		if err := req.Reqf(reqf.Rval{
			Url: "https://api.live.bilibili.com/xlive/web-room/v2/index/getRoomPlayInfo?no_playurl=0&mask=1&qn=0&platform=web&protocol=0,1&format=0,2&codec=0,1&room_id=" + Roomid,
			Header: map[string]string{
				`Referer`: "https://live.bilibili.com/" + Roomid,
				`Cookie`:  reqf.Map_2_Cookies_String(Cookie),
			},
			Proxy:   kv.LoadV(`Common.Proxy`).(string),
			Timeout: 10 * 1000,
			Retry:   2,
		}); err != nil {
			apilog.L(`E: `, err)
			return
		}

		var j J.Roominitres

		if e := json.Unmarshal([]byte(req.Respon), &j); e != nil {
			apilog.L(`E: `, e)
			return
		} else if j.Code != 0 {
			apilog.L(`E: `, j.Message)
			return
		}

		//主播uid
		kv.Store(`Common.UpUid`, j.Data.UID)
		//房间号（完整）
		if j.Data.RoomID != 0 {
			kv.Store(`Common.Roomid`, j.Data.RoomID)
		}
		//直播开始时间
		kv.Store(`Common.Live_Start_Time`, time.Unix(int64(j.Data.LiveTime), 0))
		//是否在直播
		kv.Store(`Common.Liveing`, j.Data.LiveStatus == 1)

		//未在直播，不获取直播流
		if !c.Liveing {
			kv.Store(`Common.Live_qn`, 0)
			kv.Store(`Common.AcceptQn`, kv.LoadV(`Common.Qn`).(map[int]string))
			kv.Store(`Common.Live`, []string{})
			return
		}

		//当前直播流
		{
			type Stream_name struct {
				Protocol_name string
				Format_name   string
				Codec_name    string
			}
			var name_map = map[string]Stream_name{
				`flv`: {
					Protocol_name: "http_stream",
					Format_name:   "flv",
					Codec_name:    "avc",
				},
				`hls`: {
					Protocol_name: "http_hls",
					Format_name:   "fmp4",
					Codec_name:    "avc",
				},
			}

			want_type := name_map[`hls`]
			if v, ok := kv.LoadV(`直播流类型`).(string); ok {
				if v, ok := name_map[v]; ok {
					want_type = v
				} else {
					apilog.L(`I: `, `未找到`, v, `,默认hls`)
				}
			} else {
				apilog.L(`T: `, `默认hls`)
			}
			no_found_type := true
			for {
				for _, v := range j.Data.PlayurlInfo.Playurl.Stream {
					if v.ProtocolName != want_type.Protocol_name {
						continue
					}

					for _, v := range v.Format {
						if v.FormatName != want_type.Format_name {
							continue
						}

						for _, v := range v.Codec {
							if v.CodecName != want_type.Codec_name {
								continue
							}

							//当前直播流质量
							kv.Store(`Common.Live_qn`, v.CurrentQn)
							if kv.LoadV(`Common.Live_want_qn`).(int) == 0 {
								kv.Store(`Common.Live_want_qn`, v.CurrentQn)
							}
							//允许的清晰度
							{
								var qn = kv.LoadV(`Common.Qn`).(map[int]string)
								var tmp = make(map[int]string)
								for _, v := range v.AcceptQn {
									if s, ok := qn[v]; ok {
										tmp[v] = s
									}
								}
								kv.Store(`Common.AcceptQn`, tmp)
							}
							//直播流链接
							{
								var live = []string{}
								for _, v1 := range v.URLInfo {
									live = append(live, v1.Host+v.BaseURL+v1.Extra)
								}
								kv.Store(`Common.Live`, live)
							}
						}
					}
				}
				if no_found_type {
					if want_type.Protocol_name == "http_stream" {
						apilog.L(`I: `, `不支持flv，使用hls`)
						want_type = name_map[`hls`]
					} else {
						apilog.L(`I: `, `不支持hls，使用flv`)
						want_type = name_map[`flv`]
					}
					no_found_type = false
				} else {
					break
				}
			}
		}
	}
	return
}

func getRoomPlayInfoByQn() (missKey []string) {
	apilog := apilog.Base_add(`getRoomPlayInfoByQn`)

	if c.Roomid == 0 {
		missKey = append(missKey, `Roomid`)
	}
	if !c.LIVE_BUVID {
		missKey = append(missKey, `LIVE_BUVID`)
	}
	if len(missKey) > 0 {
		return
	}

	{
		AcceptQn := []int{}
		for k, _ := range c.AcceptQn {
			if k <= c.Live_want_qn {
				AcceptQn = append(AcceptQn, k)
			}
		}
		MaxQn := 0
		for i := 0; len(AcceptQn) > i; i += 1 {
			if AcceptQn[i] > MaxQn {
				MaxQn = AcceptQn[i]
			}
		}
		if MaxQn == 0 {
			apilog.L(`W: `, "使用默认")
		}
		c.Live_qn = MaxQn
	}

	Roomid := strconv.Itoa(c.Roomid)

	//Roominitres
	{
		Cookie := make(map[string]string)
		c.Cookie.Range(func(k, v interface{}) bool {
			Cookie[k.(string)] = v.(string)
			return true
		})

		req := reqf.New()
		if err := req.Reqf(reqf.Rval{
			Url: "https://api.live.bilibili.com/xlive/web-room/v2/index/getRoomPlayInfo?no_playurl=0&mask=1&qn=" + strconv.Itoa(c.Live_qn) + "&platform=web&protocol=0,1&format=0,2&codec=0,1&room_id=" + Roomid,
			Header: map[string]string{
				`Referer`: "https://live.bilibili.com/" + Roomid,
				`Cookie`:  reqf.Map_2_Cookies_String(Cookie),
			},
			Proxy:   kv.LoadV(`Common.Proxy`).(string),
			Timeout: 10 * 1000,
			Retry:   2,
		}); err != nil {
			apilog.L(`E: `, err)
			return
		}

		var j J.Roominitres

		if e := json.Unmarshal([]byte(req.Respon), &j); e != nil {
			apilog.L(`E: `, e)
			return
		} else if j.Code != 0 {
			apilog.L(`E: `, j.Message)
			return
		}

		//主播uid
		c.UpUid = j.Data.UID
		//房间号（完整）
		if j.Data.RoomID != 0 {
			c.Roomid = j.Data.RoomID
		}
		//直播开始时间
		c.Live_Start_Time = time.Unix(int64(j.Data.LiveTime), 0)
		//是否在直播
		c.Liveing = j.Data.LiveStatus == 1

		//未在直播，不获取直播流
		if !c.Liveing {
			c.Live_qn = 0
			c.AcceptQn = c.Qn
			c.Live = []string{}
			return
		}

		//当前直播流
		{
			type Stream_name struct {
				Protocol_name string
				Format_name   string
				Codec_name    string
			}
			var name_map = map[string]Stream_name{
				`flv`: {
					Protocol_name: "http_stream",
					Format_name:   "flv",
					Codec_name:    "avc",
				},
				`hls`: {
					Protocol_name: "http_hls",
					Format_name:   "fmp4",
					Codec_name:    "avc",
				},
			}

			want_type := name_map[`hls`]
			if v, ok := c.K_v.LoadV(`直播流类型`).(string); ok {
				if v, ok := name_map[v]; ok {
					want_type = v
				} else {
					apilog.L(`I: `, `未找到`, v, `,默认hls`)
				}
			} else {
				apilog.L(`T: `, `默认hls`)
			}

			no_found_type := true
			for {
				for _, v := range j.Data.PlayurlInfo.Playurl.Stream {
					if v.ProtocolName != want_type.Protocol_name {
						continue
					}

					for _, v := range v.Format {
						if v.FormatName != want_type.Format_name {
							continue
						}

						for _, v := range v.Codec {
							if v.CodecName != want_type.Codec_name {
								continue
							}

							no_found_type = false

							//当前直播流质量
							c.Live_qn = v.CurrentQn
							if c.Live_want_qn == 0 {
								c.Live_want_qn = v.CurrentQn
							}
							//允许的清晰度
							{
								var tmp = make(map[int]string)
								for _, v := range v.AcceptQn {
									if s, ok := c.Qn[v]; ok {
										tmp[v] = s
									}
								}
								c.AcceptQn = tmp
							}
							//直播流链接
							c.Live = []string{}
							for _, v1 := range v.URLInfo {
								c.Live = append(c.Live, v1.Host+v.BaseURL+v1.Extra)
							}
						}
					}
				}
				if no_found_type {
					if want_type.Protocol_name == "http_stream" {
						apilog.L(`I: `, `不支持flv，使用hls`)
						want_type = name_map[`hls`]
					} else {
						apilog.L(`I: `, `不支持hls，使用flv`)
						want_type = name_map[`flv`]
					}
					no_found_type = false
				} else {
					break
				}
			}
			if s, ok := c.Qn[c.Live_qn]; !ok {
				apilog.L(`W: `, `未知清晰度`, c.Live_qn)
			} else {
				apilog.L(`I: `, s)
			}
		}
	}
	return
}

func getDanmuInfo() (missKey []string) {
	apilog := apilog.Base_add(`getDanmuInfo`)

	if c.Roomid == 0 {
		missKey = append(missKey, `Roomid`)
	}
	if !c.LIVE_BUVID {
		missKey = append(missKey, `LIVE_BUVID`)
	}
	if len(missKey) > 0 {
		return
	}

	Roomid := strconv.Itoa(c.Roomid)

	//GetDanmuInfo
	{
		Cookie := make(map[string]string)
		c.Cookie.Range(func(k, v interface{}) bool {
			Cookie[k.(string)] = v.(string)
			return true
		})

		req := reqf.New()
		if err := req.Reqf(reqf.Rval{
			Url: "https://api.live.bilibili.com/xlive/web-room/v1/index/getDanmuInfo?type=0&id=" + Roomid,
			Header: map[string]string{
				`Referer`: "https://live.bilibili.com/" + Roomid,
				`Cookie`:  reqf.Map_2_Cookies_String(Cookie),
			},
			Proxy:   kv.LoadV(`Common.Proxy`).(string),
			Timeout: 10 * 1000,
		}); err != nil {
			apilog.L(`E: `, err)
			return
		}

		var j J.GetDanmuInfo

		if e := json.Unmarshal([]byte(req.Respon), &j); e != nil {
			apilog.L(`E: `, e)
			return
		} else if j.Code != 0 {
			apilog.L(`E: `, j.Message)
			return
		}

		//弹幕钥
		c.Token = j.Data.Token
		//弹幕链接
		for _, v := range j.Data.HostList {
			c.WSURL = append(c.WSURL, "wss://"+v.Host+"/sub")
		}
	}
	return
}

func Get_face_src(uid string) string {
	if uid == "" {
		return ""
	}
	if api_limit.TO() {
		return ""
	} //超额请求阻塞，超时将取消
	apilog := apilog.Base_add(`获取头像`)

	Cookie := make(map[string]string)
	c.Cookie.Range(func(k, v interface{}) bool {
		Cookie[k.(string)] = v.(string)
		return true
	})

	req := reqf.New()
	if err := req.Reqf(reqf.Rval{
		Url: "https://api.live.bilibili.com/xlive/web-room/v1/index/getDanmuMedalAnchorInfo?ruid=" + uid,
		Header: map[string]string{
			`Referer`: "https://live.bilibili.com/" + strconv.Itoa(c.Roomid),
			`Cookie`:  reqf.Map_2_Cookies_String(Cookie),
		},
		Proxy:   kv.LoadV(`Common.Proxy`).(string),
		Timeout: 10 * 1000,
		Retry:   2,
	}); err != nil {
		apilog.L(`E: `, err)
		return ""
	}
	res := string(req.Respon)
	if msg := p.Json().GetValFromS(res, "message"); msg == nil || msg != "0" {
		apilog.L(`E: `, "message", msg)
		return ""
	}

	rface := p.Json().GetValFromS(res, "data.rface")
	if rface == nil {
		apilog.L(`E: `, "data.rface", rface)
		return ""
	}
	return rface.(string) + `@58w_58h`
}

func Get_HotRank() (missKey []string) {
	apilog := apilog.Base_add(`Get_HotRank`)

	if c.UpUid == 0 {
		missKey = append(missKey, `UpUid`)
	}
	if c.Roomid == 0 {
		missKey = append(missKey, `Roomid`)
	}
	if c.ParentAreaID == 0 {
		missKey = append(missKey, `ParentAreaID`)
	}
	if !c.LIVE_BUVID {
		missKey = append(missKey, `LIVE_BUVID`)
	}
	if len(missKey) > 0 {
		return
	}

	Roomid := strconv.Itoa(c.Roomid)

	//getHotRank
	{
		Cookie := make(map[string]string)
		c.Cookie.Range(func(k, v interface{}) bool {
			Cookie[k.(string)] = v.(string)
			return true
		})

		req := reqf.New()
		if err := req.Reqf(reqf.Rval{
			Url: `https://api.live.bilibili.com/xlive/general-interface/v1/rank/getHotRank?ruid=` + strconv.Itoa(c.UpUid) + `&room_id=` + Roomid + `&is_pre=0&page_size=50&source=2&area_id=` + strconv.Itoa(c.ParentAreaID),
			Header: map[string]string{
				`Host`:            `api.live.bilibili.com`,
				`User-Agent`:      `Mozilla/5.0 (X11; Linux x86_64; rv:83.0) Gecko/20100101 Firefox/83.0`,
				`Accept`:          `application/json, text/plain, */*`,
				`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
				`Accept-Encoding`: `gzip, deflate, br`,
				`Origin`:          `https://live.bilibili.com`,
				`Connection`:      `keep-alive`,
				`Pragma`:          `no-cache`,
				`Cache-Control`:   `no-cache`,
				`Referer`:         "https://live.bilibili.com/" + Roomid,
				`Cookie`:          reqf.Map_2_Cookies_String(Cookie),
			},
			Proxy:   kv.LoadV(`Common.Proxy`).(string),
			Timeout: 3 * 1000,
			Retry:   2,
		}); err != nil {
			apilog.L(`E: `, err)
			return
		}

		var j J.GetHotRank

		if e := json.Unmarshal([]byte(req.Respon), &j); e != nil {
			apilog.L(`E: `, e)
			return
		} else if j.Code != 0 {
			apilog.L(`E: `, j.Message)
			return
		}

		//获取排名
		c.Note = j.Data.Own.AreaName + " "
		if j.Data.Own.Rank == 0 {
			c.Note += "50+"
		} else {
			c.Note += strconv.Itoa(j.Data.Own.Rank)
		}
	}

	return
}

func Get_guardNum() (missKey []string) {
	apilog := apilog.Base_add(`Get_guardNum`)

	if c.UpUid == 0 {
		missKey = append(missKey, `UpUid`)
	}
	if c.Roomid == 0 {
		missKey = append(missKey, `Roomid`)
	}
	if !c.LIVE_BUVID {
		missKey = append(missKey, `LIVE_BUVID`)
	}
	if len(missKey) > 0 {
		return
	}

	Roomid := strconv.Itoa(c.Roomid)

	//Get_guardNum
	{
		Cookie := make(map[string]string)
		c.Cookie.Range(func(k, v interface{}) bool {
			Cookie[k.(string)] = v.(string)
			return true
		})

		req := reqf.New()
		if err := req.Reqf(reqf.Rval{
			Url: `https://api.live.bilibili.com/xlive/app-room/v2/guardTab/topList?roomid=` + Roomid + `&page=1&ruid=` + strconv.Itoa(c.UpUid) + `&page_size=29`,
			Header: map[string]string{
				`Host`:            `api.live.bilibili.com`,
				`User-Agent`:      `Mozilla/5.0 (X11; Linux x86_64; rv:83.0) Gecko/20100101 Firefox/83.0`,
				`Accept`:          `application/json, text/plain, */*`,
				`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
				`Accept-Encoding`: `gzip, deflate, br`,
				`Origin`:          `https://live.bilibili.com`,
				`Connection`:      `keep-alive`,
				`Pragma`:          `no-cache`,
				`Cache-Control`:   `no-cache`,
				`Referer`:         "https://live.bilibili.com/" + Roomid,
				`Cookie`:          reqf.Map_2_Cookies_String(Cookie),
			},
			Proxy:   kv.LoadV(`Common.Proxy`).(string),
			Timeout: 3 * 1000,
			Retry:   2,
		}); err != nil {
			apilog.L(`E: `, err)
			return
		}

		var j J.GetGuardNum

		if e := json.Unmarshal([]byte(req.Respon), &j); e != nil {
			apilog.L(`E: `, e)
			return
		} else if j.Code != 0 {
			apilog.L(`E: `, j.Message)
			return
		}

		//获取舰长数
		c.GuardNum = j.Data.Info.Num
	}

	return
}

// func Get_Version() (missKey []string) {  不再需要
// 	if c.Roomid == 0 {
// 		missKey = append(missKey, `Roomid`)
// 	}
// 	if len(missKey) != 0 {return}

// 	Roomid := strconv.Itoa(c.Roomid)

// 	apilog := apilog.Base_add(`获取客户版本`)

// 	var player_js_url string
// 	{//获取player_js_url
// 		r := g.Get(reqf.Rval{
// 			Url:"https://live.bilibili.com/blanc/" + Roomid,
// 		})

// 		if r.Err != nil {
// 			apilog.L(`E: `,r.Err)
// 			return
// 		}

// 		r.S2(`<script src=`,`.js`)
// 		if r.Err != nil {
// 			apilog.L(`E: `,r.Err)
// 			return
// 		}

// 		for _,v := range r.RS {
// 			tmp := string(v) + `.js`
// 			if strings.Contains(tmp,`http`) {continue}
// 			tmp = `https:` + tmp
// 			if strings.Contains(tmp,`player`) {
// 				player_js_url = tmp
// 				break
// 			}
// 		}
// 		if player_js_url == `` {
// 			apilog.L(`E: `,`no found player js`)
// 			return
// 		}
// 	}

// 	{//获取VERSION
// 		r := g.Get(reqf.Rval{
// 			Url:player_js_url,
// 		})

// 		if r.Err != nil {
// 			apilog.L(`E: `,r.Err)
// 			return
// 		}

// 		r.S(`Bilibili HTML5 Live Player v`,` `,0,0)
// 		if r.Err != nil {
// 			apilog.L(`E: `,r.Err)
// 			return
// 		}
// 		c.VERSION = r.RS[0]
// 		apilog.L(`T: `,"api version", c.VERSION)
// 	}
// 	return
// }

//调用记录
var boot_Get_cookie funcCtrl.FlashFunc //新的替代旧的

//扫码登录
func Get_cookie(kv sync.Map) (missKey []string) {
	if v, ok := kv.LoadV(`扫码登录`).(bool); !ok || !v {
		return
	}

	apilog := apilog.Base_add(`获取Cookie`)

	if p.Checkfile().IsExist("cookie.txt") { //读取cookie文件
		if cookieString := string(CookieGet()); cookieString != `` {
			//cookie存入全局变量syncmap
			kv.Store(`Cookie`, reqf.Cookies_String_2_Map(cookieString))
			if miss := CookieCheck([]string{
				`bili_jct`,
				`DedeUserID`,
			}); len(miss) == 0 {
				return
			}
		}
	}

	//获取id
	id := boot_Get_cookie.Flash()
	defer boot_Get_cookie.UnFlash()

	var img_url string
	var oauth string
	{ //获取二维码
		r := reqf.New()
		if e := r.Reqf(reqf.Rval{
			Url:     `https://passport.bilibili.com/qrcode/getLoginUrl`,
			Proxy:   kv.LoadV(`Common.Proxy`).(string),
			Timeout: 10 * 1000,
			Retry:   2,
		}); e != nil {
			apilog.L(`E: `, e)
			return
		}
		var res struct {
			Code   int  `json:"code"`
			Status bool `json:"status"`
			Data   struct {
				Url      string `json:"url"`
				OauthKey string `json:"oauthKey"`
			} `json:"data"`
		}
		if e := json.Unmarshal(r.Respon, &res); e != nil {
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
		} else {
			img_url = res.Data.Url
		}
		if res.Data.OauthKey == `` {
			apilog.L(`E: `, `Data.OauthKey == ""`)
			return
		} else {
			oauth = res.Data.OauthKey
		}
	}

	//有新实例，退出
	if boot_Get_cookie.NeedExit(id) {
		return
	}

	var server = &http.Server{
		Addr: p.Sys().GetIntranetIp() + ":" + strconv.Itoa(p.Sys().GetFreePort()),
	}
	{ //生成二维码
		qr.WriteFile(img_url, qr.Medium, 256, `qr.png`)
		if !p.Checkfile().IsExist(`qr.png`) {
			apilog.L(`E: `, `qr error`)
			return
		}
		//启动web
		s := web.New(server)
		s.Handle(map[string]func(http.ResponseWriter, *http.Request){
			`/`: func(w http.ResponseWriter, r *http.Request) {
				var path string = r.URL.Path[1:]
				if path == `` {
					path = `index.html`
				}
				http.ServeFile(w, r, path)
			},
			`/exit`: func(w http.ResponseWriter, r *http.Request) {
				s.Server.Shutdown(context.Background())
			},
		})
		defer server.Shutdown(context.Background())

		if kv.LoadV(`扫码登录自动打开标签页`).(bool) {
			open.Run(`http://` + server.Addr + `/qr.png`)
		}
		apilog.Block(1000)
		//show qr code in cmd
		qrterminal.GenerateWithConfig(img_url, qrterminal.Config{
			Level:     qrterminal.L,
			Writer:    os.Stdout,
			BlackChar: `  `,
			WhiteChar: `OO`,
		})
		apilog.L(`W: `, `手机扫命令行二维码登录`)
		apilog.L(`W: `, `或打开链接扫码登录：`, `http://`+server.Addr+`/qr.png`)
		p.Sys().Timeoutf(1)
	}

	//有新实例，退出
	if boot_Get_cookie.NeedExit(id) {
		return
	}

	var cookie string
	{ //循环查看是否通过
		Cookie := kv.LoadV(`Cookie`).(map[string]string)

		for {
			//3s刷新查看是否通过
			p.Sys().Timeoutf(3)

			//有新实例，退出
			if boot_Get_cookie.NeedExit(id) {
				return
			}

			r := reqf.New()
			if e := r.Reqf(reqf.Rval{
				Url:     `https://passport.bilibili.com/qrcode/getLoginInfo`,
				PostStr: `oauthKey=` + oauth,
				Header: map[string]string{
					`Content-Type`: `application/x-www-form-urlencoded; charset=UTF-8`,
					`Referer`:      `https://passport.bilibili.com/login`,
					`Cookie`:       reqf.Map_2_Cookies_String(Cookie),
				},
				Proxy:   kv.LoadV(`Common.Proxy`).(string),
				Timeout: 10 * 1000,
				Retry:   2,
			}); e != nil {
				apilog.L(`E: `, e)
				return
			}

			var res struct {
				Status  bool   `josn:"status"`
				Message string `json:"message"`
			}

			if e := json.Unmarshal(r.Respon, &res); e != nil {
				apilog.L(`E: `, e.Error(), string(r.Respon))
			}

			if !res.Status {
				if res.Message == `Can't Match oauthKey~` {
					apilog.L(`W: `, `登录超时`)
					return
				}
			} else {
				apilog.L(`W: `, `登录，并保存了cookie`)
				if v := r.Response.Cookies(); len(v) == 0 {
					apilog.L(`E: `, `getLoginInfo cookies len == 0`)
					return
				} else {
					cookie = reqf.Map_2_Cookies_String(reqf.Cookies_List_2_Map(v)) //cookie to string
				}
				if cookie == `` {
					apilog.L(`E: `, `getLoginInfo cookies ""`)
					return
				} else {
					break
				}
			}
		}
		if len(cookie) == 0 {
			return
		}
	}

	//有新实例，退出
	if boot_Get_cookie.NeedExit(id) {
		return
	}

	{ //写入cookie.txt
		kv.Store(`Cookie`, reqf.Cookies_String_2_Map(cookie))
		CookieSet([]byte(cookie))
	}

	//有新实例，退出
	if boot_Get_cookie.NeedExit(id) {
		return
	}

	{ //清理
		if p.Checkfile().IsExist(`qr.png`) {
			os.RemoveAll(`qr.png`)
			return
		}
	}
	return
}

//短信登录
func Get_cookie_by_msg() {
	/*https://passport.bilibili.com/x/passport-login/web/sms/send*/
}

//获取牌子信息
func Get_list_in_room() (array []J.GetMyMedals_Items) {

	apilog := apilog.Base_add(`获取牌子`)
	//验证cookie
	if missKey := CookieCheck([]string{
		`bili_jct`,
		`DedeUserID`,
		`LIVE_BUVID`,
	}); len(missKey) != 0 {
		apilog.L(`T: `, `Cookie无Key:`, missKey)
		return
	}
	Cookie := make(map[string]string)
	c.Cookie.Range(func(k, v interface{}) bool {
		Cookie[k.(string)] = v.(string)
		return true
	})

	{ //获取牌子列表
		var medalList []J.GetMyMedals_Items
		for pageNum := 1; true; pageNum += 1 {
			r := reqf.New()
			if e := r.Reqf(reqf.Rval{
				Url: `https://api.live.bilibili.com/xlive/app-ucenter/v1/user/GetMyMedals?page=` + strconv.Itoa(pageNum) + `&page_size=10`,
				Header: map[string]string{
					`Cookie`: reqf.Map_2_Cookies_String(Cookie),
				},
				Proxy:   kv.LoadV(`Common.Proxy`).(string),
				Timeout: 10 * 1000,
				Retry:   2,
			}); e != nil {
				apilog.L(`E: `, e)
				return
			}

			var res J.GetMyMedals

			if e := json.Unmarshal(r.Respon, &res); e != nil {
				apilog.L(`E: `, e)
			}

			if res.Code != 0 {
				apilog.L(`E: `, `返回code`, res.Code, res.Message)
				return
			}

			medalList = append(medalList, res.Data.Items...)

			if res.Data.PageInfo.CurPage == res.Data.PageInfo.TotalPage {
				break
			}

			time.Sleep(time.Second)
		}

		return medalList
	}
}

//获取当前佩戴的牌子
func Get_weared_medal() (item J.GetWearedMedal_Data) {

	apilog := apilog.Base_add(`获取牌子`)
	//验证cookie
	if missKey := CookieCheck([]string{
		`bili_jct`,
		`DedeUserID`,
		`LIVE_BUVID`,
	}); len(missKey) != 0 {
		apilog.L(`T: `, `Cookie无Key:`, missKey)
		return
	}
	Cookie := make(map[string]string)
	c.Cookie.Range(func(k, v interface{}) bool {
		Cookie[k.(string)] = v.(string)
		return true
	})

	{ //获取
		r := reqf.New()
		if e := r.Reqf(reqf.Rval{
			Url: `https://api.live.bilibili.com/live_user/v1/UserInfo/get_weared_medal`,
			Header: map[string]string{
				`Cookie`: reqf.Map_2_Cookies_String(Cookie),
			},
			Proxy:   kv.LoadV(`Common.Proxy`).(string),
			Timeout: 10 * 1000,
			Retry:   2,
		}); e != nil {
			apilog.L(`E: `, e)
			return
		}

		var res J.GetWearedMedal
		if e := json.Unmarshal(r.Respon, &res); e != nil {
			apilog.L(`E: `, e)
			return
		}

		if res.Code != 0 {
			apilog.L(`E: `, `返回code`, res.Code, res.Msg)
			return
		}

		switch res.Data.(type) {
		case J.GetWearedMedal_Data:
			return res.Data.(J.GetWearedMedal_Data)
		default:
		}
		return
	}

}

func CheckSwitch_FansMedal() (missKey []string) {

	if !c.LIVE_BUVID {
		missKey = append(missKey, `LIVE_BUVID`)
	}
	if c.UpUid == 0 {
		missKey = append(missKey, `UpUid`)
	}
	if len(missKey) > 0 {
		return
	}

	apilog := apilog.Base_add(`切换粉丝牌`)
	//验证cookie
	if missCookie := CookieCheck([]string{
		`bili_jct`,
		`DedeUserID`,
		`LIVE_BUVID`,
	}); len(missCookie) != 0 {
		apilog.L(`T: `, `Cookie无Key:`, missCookie)
		return
	}

	Cookie := make(map[string]string)
	c.Cookie.Range(func(k, v interface{}) bool {
		Cookie[k.(string)] = v.(string)
		return true
	})
	{ //获取当前牌子，验证是否本直播间牌子
		res := Get_weared_medal()

		c.Wearing_FansMedal = res.Roominfo.RoomID //更新佩戴信息
		if res.TargetID == c.UpUid {
			return
		}
	}

	var medal_id int //将要使用的牌子id
	//检查是否有此直播间的牌子
	{
		medal_list := Get_list_in_room()
		for _, v := range medal_list {
			if v.TargetID != c.UpUid {
				continue
			}
			medal_id = v.MedalID
		}
		if medal_id == 0 { //无牌
			apilog.L(`I: `, `无主播粉丝牌`)
			if c.Wearing_FansMedal == 0 { //当前没牌
				return
			}
		}
	}

	var (
		post_url string
		post_str string
	)
	{ //生成佩戴信息
		csrf, _ := c.Cookie.LoadV(`bili_jct`).(string)
		if csrf == `` {
			apilog.L(`E: `, "Cookie错误,无bili_jct=")
			return
		}

		post_str = `csrf_token=` + csrf + `&csrf=` + csrf

		if medal_id == 0 { //无牌，不佩戴牌子
			post_url = `https://api.live.bilibili.com/xlive/web-room/v1/fansMedal/take_off`
		} else {
			post_url = `https://api.live.bilibili.com/xlive/web-room/v1/fansMedal/wear`
			post_str = `medal_id=` + strconv.Itoa(medal_id) + `&` + post_str
		}
	}
	{ //切换牌子
		r := reqf.New()
		if e := r.Reqf(reqf.Rval{
			Url:     post_url,
			PostStr: post_str,
			Header: map[string]string{
				`Cookie`:       reqf.Map_2_Cookies_String(Cookie),
				`Content-Type`: `application/x-www-form-urlencoded; charset=UTF-8`,
				`Referer`:      `https://passport.bilibili.com/login`,
			},
			Proxy:   kv.LoadV(`Common.Proxy`).(string),
			Timeout: 10 * 1000,
			Retry:   2,
		}); e != nil {
			apilog.L(`E: `, e)
			return
		}
		res := string(r.Respon)
		if v, ok := p.Json().GetValFromS(res, "code").(float64); ok && v == 0 {
			apilog.L(`I: `, `自动切换粉丝牌 id:`, medal_id)
			c.Wearing_FansMedal = medal_id //更新佩戴信息
			return
		}
		if v, ok := p.Json().GetValFromS(res, "message").(string); ok {
			apilog.L(`E: `, `Get_FansMedal wear message`, v)
		} else {
			apilog.L(`E: `, `Get_FansMedal wear message nil`)
		}
	}
	return
}

//签到
func Dosign() {
	apilog := apilog.Base_add(`签到`).L(`T: `, `签到`)
	//验证cookie
	if missKey := CookieCheck([]string{
		`bili_jct`,
		`DedeUserID`,
		`LIVE_BUVID`,
	}); len(missKey) != 0 {
		apilog.L(`T: `, `Cookie无Key:`, missKey)
		return
	}

	{ //检查是否签到
		Cookie := make(map[string]string)
		c.Cookie.Range(func(k, v interface{}) bool {
			Cookie[k.(string)] = v.(string)
			return true
		})

		req := reqf.New()
		if err := req.Reqf(reqf.Rval{
			Url: `https://api.live.bilibili.com/xlive/web-ucenter/v1/sign/WebGetSignInfo`,
			Header: map[string]string{
				`Host`:            `api.live.bilibili.com`,
				`User-Agent`:      `Mozilla/5.0 (X11; Linux x86_64; rv:83.0) Gecko/20100101 Firefox/83.0`,
				`Accept`:          `application/json, text/plain, */*`,
				`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
				`Accept-Encoding`: `gzip, deflate, br`,
				`Origin`:          `https://live.bilibili.com`,
				`Connection`:      `keep-alive`,
				`Pragma`:          `no-cache`,
				`Cache-Control`:   `no-cache`,
				`Referer`:         "https://live.bilibili.com/all",
				`Cookie`:          reqf.Map_2_Cookies_String(Cookie),
			},
			Proxy:   kv.LoadV(`Common.Proxy`).(string),
			Timeout: 3 * 1000,
			Retry:   2,
		}); err != nil {
			apilog.L(`E: `, err)
			return
		}

		var msg struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    struct {
				Status int `json:"status"`
			} `json:"data"`
		}
		if e := json.Unmarshal(req.Respon, &msg); e != nil {
			apilog.L(`E: `, e)
		}
		if msg.Code != 0 {
			apilog.L(`E: `, msg.Message)
			return
		}
		if msg.Data.Status == 1 { //今日已签到
			return
		}
	}

	{ //签到
		Cookie := make(map[string]string)
		c.Cookie.Range(func(k, v interface{}) bool {
			Cookie[k.(string)] = v.(string)
			return true
		})

		req := reqf.New()
		if err := req.Reqf(reqf.Rval{
			Url: `https://api.live.bilibili.com/xlive/web-ucenter/v1/sign/DoSign`,
			Header: map[string]string{
				`Host`:            `api.live.bilibili.com`,
				`User-Agent`:      `Mozilla/5.0 (X11; Linux x86_64; rv:83.0) Gecko/20100101 Firefox/83.0`,
				`Accept`:          `application/json, text/plain, */*`,
				`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
				`Accept-Encoding`: `gzip, deflate, br`,
				`Origin`:          `https://live.bilibili.com`,
				`Connection`:      `keep-alive`,
				`Pragma`:          `no-cache`,
				`Cache-Control`:   `no-cache`,
				`Referer`:         "https://live.bilibili.com/all",
				`Cookie`:          reqf.Map_2_Cookies_String(Cookie),
			},
			Proxy:   kv.LoadV(`Common.Proxy`).(string),
			Timeout: 3 * 1000,
			Retry:   2,
		}); err != nil {
			apilog.L(`E: `, err)
			return
		}

		var msg struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    struct {
				HadSignDays int `json:"hadSignDays"`
			} `json:"data"`
		}
		if e := json.Unmarshal(req.Respon, &msg); e != nil {
			apilog.L(`E: `, e)
		}
		if msg.Code == 0 {
			apilog.L(`I: `, `签到成功!本月已签到`, msg.Data.HadSignDays, `天`)
			return
		}
		apilog.L(`E: `, msg.Message)
	}
}

//LIVE_BUVID
func Get_LIVE_BUVID() (missKey []string) {
	apilog := apilog.Base_add(`LIVE_BUVID`).L(`T: `, `获取`)

	if live_buvid, ok := c.Cookie.LoadV(`LIVE_BUVID`).(string); ok && live_buvid != `` {
		apilog.L(`T: `, `存在`)
		c.LIVE_BUVID = true
		return
	}

	//当房间处于特殊活动状态时，将会获取不到，此处使用了若干著名up主房间进行尝试
	roomIdList := []string{
		"3", //哔哩哔哩音悦台
		"2", //直播姬
		"1", //哔哩哔哩直播
	}

	for _, roomid := range roomIdList { //获取
		req := reqf.New()
		if err := req.Reqf(reqf.Rval{
			Url: `https://api.live.bilibili.com/live/getRoomKanBanModel?roomid=` + roomid,
			Header: map[string]string{
				`Host`:                      `live.bilibili.com`,
				`User-Agent`:                `Mozilla/5.0 (X11; Linux x86_64; rv:83.0) Gecko/20100101 Firefox/83.0`,
				`Accept`:                    `text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8`,
				`Accept-Language`:           `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
				`Accept-Encoding`:           `gzip, deflate, br`,
				`Connection`:                `keep-alive`,
				`Cache-Control`:             `no-cache`,
				`Referer`:                   "https://live.bilibili.com",
				`DNT`:                       `1`,
				`Upgrade-Insecure-Requests`: `1`,
			},
			Proxy:   kv.LoadV(`Common.Proxy`).(string),
			Timeout: 3 * 1000,
			Retry:   2,
		}); err != nil {
			apilog.L(`E: `, err)
			return
		}

		//cookie
		var has bool
		for k, v := range reqf.Cookies_List_2_Map(req.Response.Cookies()) {
			c.Cookie.Store(k, v)
			if k == `LIVE_BUVID` {
				has = true
			}
		}
		if has {
			apilog.L(`I: `, `获取到LIVE_BUVID，保存cookie`)
			break
		} else {
			apilog.L(`I: `, roomid, `未获取到，重试`)
			time.Sleep(time.Second)
		}
	}

	Cookie := make(map[string]string)
	c.Cookie.Range(func(k, v interface{}) bool {
		Cookie[k.(string)] = v.(string)
		return true
	})

	CookieSet([]byte(reqf.Map_2_Cookies_String(Cookie)))

	c.LIVE_BUVID = true

	return
}

//小心心
type E_json struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Ttl     int    `json:"ttl"`
	Data    struct {
		Timestamp          int    `json:"timestamp"`
		Heartbeat_interval int    `json:"heartbeat_interval"`
		Secret_key         string `json:"secret_key"`
		Secret_rule        []int  `json:"secret_rule"`
		Patch_status       int    `json:"patch_status"`
	} `json:"data"`
}

//调用记录
var boot_F_x25Kn funcCtrl.FlashFunc //新的替代旧的

func F_x25Kn_cancel() {
	apilog.Base_add(`小心心`).L(`T: `, `取消`)
	boot_F_x25Kn.Flash() //获取函数调用会话id
	boot_F_x25Kn.UnFlash()
}

func F_x25Kn() {
	apilog := apilog.Base_add(`小心心`)
	if c.Wearing_FansMedal == 0 {
		apilog.L(`I: `, `无粉丝牌，不获取`)
		return
	}
	//验证cookie
	if missKey := CookieCheck([]string{
		`bili_jct`,
		`DedeUserID`,
		`LIVE_BUVID`,
	}); len(missKey) != 0 {
		apilog.L(`T: `, `Cookie无Key:`, missKey)
		return
	}
	if c.ParentAreaID == -1 {
		apilog.L(`E: `, `失败！未获取Parent_area_id`)
		return
	}
	if c.AreaID == -1 {
		apilog.L(`E: `, `失败！未获取Area_id`)
		return
	}
	if api_limit.TO() {
		apilog.L(`E: `, `超时！`)
		return
	} //超额请求阻塞，超时将取消

	id := boot_F_x25Kn.Flash() //获取函数调用会话id
	defer boot_F_x25Kn.UnFlash()

	{ //查看今天小心心数量
		var num = 0
		for _, v := range Gift_list() {
			if v.Gift_id == 30607 && v.Expire_at-int(p.Sys().GetSTime()) > 6*86400 {
				num = v.Gift_num
			}
		}
		if num == 24 {
			Close(0) //关闭全部（0）浏览器websocket连接
			apilog.L(`I: `, `今天小心心已满！`)
			return
		} else {
			apilog.L(`I: `, `今天已有`, num, `个小心心，开始获取`)
			defer apilog.L(`T: `, `退出`)
		}
	}

	var (
		res      E_json
		loop_num = 0
	)

	csrf, _ := c.Cookie.LoadV(`bili_jct`).(string)
	if csrf == `` {
		apilog.L(`E: `, "Cookie错误,无bili_jct")
		return
	}

	LIVE_BUVID := c.Cookie.LoadV(`LIVE_BUVID`).(string)
	if LIVE_BUVID == `` {
		apilog.L(`E: `, "Cookie错误,无LIVE_BUVID")
		return
	}

	var new_uuid string
	{
		if tmp_uuid, e := uuid.NewV4(); e == nil {
			new_uuid = tmp_uuid.String()
		} else {
			apilog.L(`E: `, e)
			return
		}
	}

	{ //初始化

		PostStr := `id=[` + strconv.Itoa(c.ParentAreaID) + `,` + strconv.Itoa(c.AreaID) + `,` + strconv.Itoa(loop_num) + `,` + strconv.Itoa(c.Roomid) + `]&`
		PostStr += `device=["` + LIVE_BUVID + `","` + new_uuid + `"]&`
		PostStr += `ts=` + strconv.Itoa(int(p.Sys().GetMTime()))
		PostStr += `&is_patch=0&`
		PostStr += `heart_beat=[]&`
		PostStr += `ua=Mozilla/5.0 (X11; Linux x86_64; rv:84.0) Gecko/20100101 Firefox/84.0&`
		PostStr += `csrf_token=` + csrf + `&csrf=` + csrf + `&`
		PostStr += `visit_id=`

		Cookie := make(map[string]string)
		c.Cookie.Range(func(k, v interface{}) bool {
			Cookie[k.(string)] = v.(string)
			return true
		})

		req := reqf.New()
		for {
			//新调用，此退出
			if boot_F_x25Kn.NeedExit(id) {
				return
			}

			if err := req.Reqf(reqf.Rval{
				Url: `https://live-trace.bilibili.com/xlive/data-interface/v1/x25Kn/E`,
				Header: map[string]string{
					`Host`:            `api.live.bilibili.com`,
					`User-Agent`:      `Mozilla/5.0 (X11; Linux x86_64; rv:83.0) Gecko/20100101 Firefox/83.0`,
					`Accept`:          `application/json, text/plain, */*`,
					`Content-Type`:    `application/x-www-form-urlencoded`,
					`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
					`Accept-Encoding`: `gzip, deflate, br`,
					`Origin`:          `https://live.bilibili.com`,
					`Connection`:      `keep-alive`,
					`Pragma`:          `no-cache`,
					`Cache-Control`:   `no-cache`,
					`Referer`:         "https://live.bilibili.com/" + strconv.Itoa(c.Roomid),
					`Cookie`:          reqf.Map_2_Cookies_String(Cookie),
				},
				PostStr: url.PathEscape(PostStr),
				Proxy:   kv.LoadV(`Common.Proxy`).(string),
				Timeout: 5 * 1000,
				Retry:   2,
			}); err != nil {
				if !reqf.IsTimeout(err) {
					apilog.L(`E: `, err)
					return
				}
				apilog.L(`W: `, `响应超时，1min后重试`)
				time.Sleep(time.Minute)
			} else {
				break
			}
		}

		if e := json.Unmarshal(req.Respon, &res); e != nil {
			apilog.L(`E: `, e)
			return
		}

		if res.Code != 0 {
			apilog.L(`E: `, `返回错误`, res.Message)
			return
		}
	}

	{ //loop
		for loop_num < (24+2)*5 {
			loop_num += 1
			//查看今天小心心数量
			if loop_num > 5 && loop_num%5 == 2 { //5min后每5min
				{ //查看今天小心心数量
					var num = 0
					for _, v := range Gift_list() {
						if v.Gift_id == 30607 && v.Expire_at-int(p.Sys().GetSTime()) > 6*86400 {
							num = v.Gift_num
						}
					}
					if num == 24 {
						Close(0) //关闭全部（0）浏览器websocket连接
						apilog.L(`I: `, `今天小心心已满！`)
						return
					} else {
						apilog.L(`I: `, `获取了今天的第`, num, `个小心心`)
					}
				}
			}

			<-time.After(time.Second * time.Duration(res.Data.Heartbeat_interval))

			//新调用，此退出
			if boot_F_x25Kn.NeedExit(id) {
				return
			}

			var (
				rt_obj = RT{
					R: R{
						Id:        `[` + strconv.Itoa(c.ParentAreaID) + `,` + strconv.Itoa(c.AreaID) + `,` + strconv.Itoa(loop_num) + `,` + strconv.Itoa(c.Roomid) + `]`,
						Device:    `["` + LIVE_BUVID + `","` + new_uuid + `"]`,
						Ets:       res.Data.Timestamp,
						Benchmark: res.Data.Secret_key,
						Time:      res.Data.Heartbeat_interval,
						Ts:        int(p.Sys().GetMTime()),
						Ua:        `Mozilla/5.0 (X11; Linux x86_64; rv:84.0) Gecko/20100101 Firefox/84.0`,
					},
					T: res.Data.Secret_rule,
				}
				wasm string
			)

			if rt_obj, wasm = Wasm(0, rt_obj); wasm == `` { //0全局
				apilog.L(`E: `, `发生错误`)
				return
			}

			PostStr := `id=` + rt_obj.R.Id + `&`
			PostStr += `device=["` + LIVE_BUVID + `","` + new_uuid + `"]&`
			PostStr += `ets=` + strconv.Itoa(res.Data.Timestamp)
			PostStr += `&benchmark=` + res.Data.Secret_key
			PostStr += `&time=` + strconv.Itoa(res.Data.Heartbeat_interval)
			PostStr += `&ts=` + strconv.Itoa(rt_obj.R.Ts)
			PostStr += `&is_patch=0&`
			PostStr += `heart_beat=[]&`
			PostStr += `ua=` + rt_obj.R.Ua + `&`
			PostStr += `csrf_token=` + csrf + `&csrf=` + csrf + `&`
			PostStr += `visit_id=`
			PostStr = `s=` + wasm + `&` + PostStr

			Cookie := make(map[string]string)
			c.Cookie.Range(func(k, v interface{}) bool {
				Cookie[k.(string)] = v.(string)
				return true
			})

			req := reqf.New()
			if err := req.Reqf(reqf.Rval{
				Url: `https://live-trace.bilibili.com/xlive/data-interface/v1/x25Kn/X`,
				Header: map[string]string{
					`Host`:            `api.live.bilibili.com`,
					`User-Agent`:      `Mozilla/5.0 (X11; Linux x86_64; rv:83.0) Gecko/20100101 Firefox/83.0`,
					`Accept`:          `application/json, text/plain, */*`,
					`Content-Type`:    `application/x-www-form-urlencoded`,
					`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
					`Accept-Encoding`: `gzip, deflate, br`,
					`Origin`:          `https://live.bilibili.com`,
					`Connection`:      `keep-alive`,
					`Pragma`:          `no-cache`,
					`Cache-Control`:   `no-cache`,
					`Referer`:         "https://live.bilibili.com/" + strconv.Itoa(c.Roomid),
					`Cookie`:          reqf.Map_2_Cookies_String(Cookie),
				},
				PostStr: url.PathEscape(PostStr),
				Proxy:   kv.LoadV(`Common.Proxy`).(string),
				Timeout: 5 * 1000,
				Retry:   2,
			}); err != nil {
				if !reqf.IsTimeout(err) {
					loop_num -= 1
					apilog.L(`W: `, `响应超时，将重试`)
					continue
				}
				apilog.L(`E: `, err)
				return
			}

			if e := json.Unmarshal(req.Respon, &res); e != nil {
				apilog.L(`E: `, e)
				return
			}

			if res.Code != 0 {
				apilog.L(`E: `, `返回错误`, res.Message)
				return
			}
		}
	}
}

//礼物列表
type Gift_list_type struct {
	Code    int                 `json:"code"`
	Message string              `json:"message"`
	Data    Gift_list_type_Data `json:"data"`
}

type Gift_list_type_Data struct {
	List []Gift_list_type_Data_List `json:"list"`
}

type Gift_list_type_Data_List struct {
	Bag_id    int    `json:"bag_id"`
	Gift_id   int    `json:"gift_id"`
	Gift_name string `json:"gift_name"`
	Gift_num  int    `json:"gift_num"`
	Expire_at int    `json:"expire_at"`
}

func Gift_list() (list []Gift_list_type_Data_List) {
	apilog := apilog.Base_add(`礼物列表`)
	//验证cookie
	if missKey := CookieCheck([]string{
		`bili_jct`,
		`DedeUserID`,
		`LIVE_BUVID`,
	}); len(missKey) != 0 {
		apilog.L(`T: `, `Cookie无Key:`, missKey)
		return
	}
	if c.Roomid == 0 {
		apilog.L(`E: `, `失败！无Roomid`)
		return
	}
	if api_limit.TO() {
		apilog.L(`E: `, `超时！`)
		return
	} //超额请求阻塞，超时将取消

	Cookie := make(map[string]string)
	c.Cookie.Range(func(k, v interface{}) bool {
		Cookie[k.(string)] = v.(string)
		return true
	})

	req := reqf.New()
	if err := req.Reqf(reqf.Rval{
		Url: `https://api.live.bilibili.com/xlive/web-room/v1/gift/bag_list?t=` + strconv.Itoa(int(p.Sys().GetMTime())) + `&room_id=` + strconv.Itoa(c.Roomid),
		Header: map[string]string{
			`Host`:            `api.live.bilibili.com`,
			`User-Agent`:      `Mozilla/5.0 (X11; Linux x86_64; rv:83.0) Gecko/20100101 Firefox/83.0`,
			`Accept`:          `application/json, text/plain, */*`,
			`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
			`Accept-Encoding`: `gzip, deflate, br`,
			`Origin`:          `https://live.bilibili.com`,
			`Connection`:      `keep-alive`,
			`Pragma`:          `no-cache`,
			`Cache-Control`:   `no-cache`,
			`Referer`:         "https://live.bilibili.com/" + strconv.Itoa(c.Roomid),
			`Cookie`:          reqf.Map_2_Cookies_String(Cookie),
		},
		Proxy:   kv.LoadV(`Common.Proxy`).(string),
		Timeout: 3 * 1000,
		Retry:   2,
	}); err != nil {
		apilog.L(`E: `, err)
		return
	}

	var res Gift_list_type

	if e := json.Unmarshal(req.Respon, &res); e != nil {
		apilog.L(`E: `, e)
		return
	}

	if res.Code != 0 {
		apilog.L(`E: `, res.Message)
		return
	}

	apilog.L(`T: `, `成功`)
	return res.Data.List
}

//银瓜子2硬币
func Silver_2_coin() (missKey []string) {
	apilog := apilog.Base_add(`银瓜子=>硬币`)

	if !c.LIVE_BUVID {
		missKey = append(missKey, `LIVE_BUVID`)
	}
	if len(missKey) > 0 {
		return
	}

	//验证cookie
	if miss := CookieCheck([]string{
		`bili_jct`,
		`DedeUserID`,
		`LIVE_BUVID`,
	}); len(miss) != 0 {
		apilog.L(`T: `, `Cookie无Key:`, miss)
		return
	}

	var Silver int
	{ //验证是否还有机会
		Cookie := make(map[string]string)
		c.Cookie.Range(func(k, v interface{}) bool {
			Cookie[k.(string)] = v.(string)
			return true
		})

		req := reqf.New()
		if err := req.Reqf(reqf.Rval{
			Url: `https://api.live.bilibili.com/xlive/revenue/v1/wallet/getStatus`,
			Header: map[string]string{
				`Host`:            `api.live.bilibili.com`,
				`User-Agent`:      `Mozilla/5.0 (X11; Linux x86_64; rv:83.0) Gecko/20100101 Firefox/83.0`,
				`Accept`:          `application/json, text/plain, */*`,
				`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
				`Accept-Encoding`: `gzip, deflate, br`,
				`Origin`:          `https://link.bilibili.com`,
				`Connection`:      `keep-alive`,
				`Pragma`:          `no-cache`,
				`Cache-Control`:   `no-cache`,
				`Referer`:         `https://link.bilibili.com/p/center/index`,
				`Cookie`:          reqf.Map_2_Cookies_String(Cookie),
			},
			Proxy:   kv.LoadV(`Common.Proxy`).(string),
			Timeout: 3 * 1000,
			Retry:   2,
		}); err != nil {
			apilog.L(`E: `, err)
			return
		}

		var j J.ApiXliveRevenueV1WalletGetStatus

		if e := json.Unmarshal([]byte(req.Respon), &j); e != nil {
			apilog.L(`E: `, e)
			return
		} else if j.Code != 0 {
			apilog.L(`E: `, j.Message)
			return
		}

		if j.Data.Silver2CoinLeft == 0 {
			apilog.L(`I: `, `今天次数已用完`)
			return
		}

		apilog.L(`T: `, `现在有银瓜子`, j.Data.Silver, `个`)
		Silver = j.Data.Silver
	}

	{ //获取交换规则，验证数量足够
		Cookie := make(map[string]string)
		c.Cookie.Range(func(k, v interface{}) bool {
			Cookie[k.(string)] = v.(string)
			return true
		})

		req := reqf.New()
		if err := req.Reqf(reqf.Rval{
			Url: `https://api.live.bilibili.com/xlive/revenue/v1/wallet/getRule`,
			Header: map[string]string{
				`Host`:            `api.live.bilibili.com`,
				`User-Agent`:      `Mozilla/5.0 (X11; Linux x86_64; rv:83.0) Gecko/20100101 Firefox/83.0`,
				`Accept`:          `application/json, text/plain, */*`,
				`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
				`Accept-Encoding`: `gzip, deflate, br`,
				`Origin`:          `https://link.bilibili.com`,
				`Connection`:      `keep-alive`,
				`Pragma`:          `no-cache`,
				`Cache-Control`:   `no-cache`,
				`Referer`:         `https://link.bilibili.com/p/center/index`,
				`Cookie`:          reqf.Map_2_Cookies_String(Cookie),
			},
			Proxy:   kv.LoadV(`Common.Proxy`).(string),
			Timeout: 3 * 1000,
			Retry:   2,
		}); err != nil {
			apilog.L(`E: `, err)
			return
		}

		var j J.ApixliveRevenueV1WalletGetRule

		if e := json.Unmarshal([]byte(req.Respon), &j); e != nil {
			apilog.L(`E: `, e)
			return
		} else if j.Code != 0 {
			apilog.L(`E: `, j.Message)
			return
		}

		if Silver < j.Data.Silver2CoinPrice {
			apilog.L(`W: `, `当前银瓜子数量不足`)
			return
		}
	}

	{ //交换
		csrf, _ := c.Cookie.LoadV(`bili_jct`).(string)
		if csrf == `` {
			apilog.L(`E: `, "Cookie错误,无bili_jct=")
			return
		}

		post_str := `csrf_token=` + csrf + `&csrf=` + csrf

		Cookie := make(map[string]string)
		c.Cookie.Range(func(k, v interface{}) bool {
			Cookie[k.(string)] = v.(string)
			return true
		})

		req := reqf.New()
		if err := req.Reqf(reqf.Rval{
			Url:     `https://api.live.bilibili.com/xlive/revenue/v1/wallet/silver2coin`,
			PostStr: url.PathEscape(post_str),
			Header: map[string]string{
				`Host`:            `api.live.bilibili.com`,
				`User-Agent`:      `Mozilla/5.0 (X11; Linux x86_64; rv:83.0) Gecko/20100101 Firefox/83.0`,
				`Accept`:          `application/json, text/plain, */*`,
				`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
				`Accept-Encoding`: `gzip, deflate, br`,
				`Origin`:          `https://link.bilibili.com`,
				`Connection`:      `keep-alive`,
				`Pragma`:          `no-cache`,
				`Cache-Control`:   `no-cache`,
				`Content-Type`:    `application/x-www-form-urlencoded`,
				`Referer`:         `https://link.bilibili.com/p/center/index`,
				`Cookie`:          reqf.Map_2_Cookies_String(Cookie),
			},
			Proxy:   kv.LoadV(`Common.Proxy`).(string),
			Timeout: 3 * 1000,
			Retry:   2,
		}); err != nil {
			apilog.L(`E: `, err)
			return
		}

		save_cookie(req.Response.Cookies())

		var res struct {
			Code    int    `json:"code"`
			Msg     string `json:"msg"`
			Message string `json:"message"`
		}

		if e := json.Unmarshal(req.Respon, &res); e != nil {
			apilog.L(`E: `, e)
			return
		}

		if res.Code != 0 {
			apilog.L(`E: `, res.Message)
			return
		}
		apilog.L(`I: `, res.Message)
	}
	return
}

func save_cookie(Cookies []*http.Cookie) {
	for k, v := range reqf.Cookies_List_2_Map(Cookies) {
		c.Cookie.Store(k, v)
	}

	Cookie := make(map[string]string)
	c.Cookie.Range(func(k, v interface{}) bool {
		Cookie[k.(string)] = v.(string)
		return true
	})
	CookieSet([]byte(reqf.Map_2_Cookies_String(Cookie)))
}

//正在直播主播
type UpItem struct {
	Uname  string `json:"uname"`
	Title  string `json:"title"`
	Roomid int    `json:"roomid"`
}

func Feed_list() (Uplist []UpItem) {
	apilog := apilog.Base_add(`正在直播主播`).L(`T: `, `获取中`)
	//验证cookie
	if missKey := CookieCheck([]string{
		`bili_jct`,
		`DedeUserID`,
		`LIVE_BUVID`,
	}); len(missKey) != 0 {
		apilog.L(`T: `, `Cookie无Key:`, missKey)
		return
	}
	if api_limit.TO() {
		apilog.L(`E: `, `超时！`)
		return
	} //超额请求阻塞，超时将取消

	Cookie := make(map[string]string)
	c.Cookie.Range(func(k, v interface{}) bool {
		Cookie[k.(string)] = v.(string)
		return true
	})

	req := reqf.New()
	for pageNum := 1; true; pageNum += 1 {
		if err := req.Reqf(reqf.Rval{
			Url: `https://api.live.bilibili.com/xlive/web-ucenter/v1/xfetter/FeedList?page=` + strconv.Itoa(pageNum) + `&pagesize=10`,
			Header: map[string]string{
				`Host`:            `api.live.bilibili.com`,
				`User-Agent`:      `Mozilla/5.0 (X11; Linux x86_64; rv:83.0) Gecko/20100101 Firefox/83.0`,
				`Accept`:          `application/json, text/plain, */*`,
				`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
				`Accept-Encoding`: `gzip, deflate, br`,
				`Origin`:          `https://t.bilibili.com`,
				`Connection`:      `keep-alive`,
				`Pragma`:          `no-cache`,
				`Cache-Control`:   `no-cache`,
				`Referer`:         `https://t.bilibili.com/pages/nav/index_new`,
				`Cookie`:          reqf.Map_2_Cookies_String(Cookie),
			},
			Proxy:   kv.LoadV(`Common.Proxy`).(string),
			Timeout: 3 * 1000,
			Retry:   2,
		}); err != nil {
			apilog.L(`E: `, err)
			return
		}

		var res struct {
			Code    int    `json:"code"`
			Msg     string `json:"msg"`
			Message string `json:"message"`
			Data    struct {
				Results int      `json:"results"`
				List    []UpItem `json:"list"`
			} `json:"data"`
		}

		if e := json.Unmarshal(req.Respon, &res); e != nil {
			apilog.L(`E: `, e)
			return
		}

		if res.Code != 0 {
			apilog.L(`E: `, res.Message)
			return
		}

		Uplist = append(Uplist, res.Data.List...)

		if pageNum*10 > res.Data.Results {
			break
		}
		time.Sleep(time.Second)
	}

	apilog.L(`T: `, `完成`)
	return
}

func GetHistory(Roomid_int int) (j J.GetHistory) {
	apilog := apilog.Base_add(`GetHistory`)

	Roomid := strconv.Itoa(Roomid_int)

	{ //使用其他api
		req := reqf.New()
		if err := req.Reqf(reqf.Rval{
			Url: "https://api.live.bilibili.com/xlive/web-room/v1/dM/gethistory?roomid=" + Roomid,
			Header: map[string]string{
				`Referer`: "https://live.bilibili.com/" + Roomid,
			},
			Proxy:   kv.LoadV(`Common.Proxy`).(string),
			Timeout: 10 * 1000,
			Retry:   2,
		}); err != nil {
			apilog.L(`E: `, err)
			return
		}

		//GetHistory
		{
			if e := json.Unmarshal(req.Respon, &j); e != nil {
				apilog.L(`E: `, e)
				return
			} else if j.Code != 0 {
				apilog.L(`E: `, j.Message)
				return
			}
		}
	}
	return
}

type searchresult struct {
	Roomid  int
	Uname   string
	Is_live bool
}

func SearchUP(s string) (list []searchresult) {
	apilog := apilog.Base_add(`搜索主播`)
	if api_limit.TO() {
		apilog.L(`E: `, `超时！`)
		return
	} //超额请求阻塞，超时将取消

	{ //使用其他api
		req := reqf.New()
		if err := req.Reqf(reqf.Rval{
			Url:     "https://api.bilibili.com/x/web-interface/search/type?context=&search_type=live_user&cover_type=user_cover&page=1&order=&category_id=&__refresh__=true&_extra=&highlight=1&single_column=0&keyword=" + url.PathEscape(s),
			Proxy:   kv.LoadV(`Common.Proxy`).(string),
			Timeout: 10 * 1000,
			Retry:   2,
		}); err != nil {
			apilog.L(`E: `, err)
			return
		}

		var j J.Search

		//Search
		{
			if e := json.Unmarshal(req.Respon, &j); e != nil {
				apilog.L(`E: `, e)
				return
			} else if j.Code != 0 {
				apilog.L(`E: `, j.Message)
				return
			}
		}

		if j.Data.Numresults == 0 {
			apilog.L(`I: `, `没有匹配`)
			return
		}

		for i := 0; i < len(j.Data.Result); i += 1 {
			uname := strings.ReplaceAll(j.Data.Result[i].Uname, `<em class="keyword">`, ``)
			uname = strings.ReplaceAll(uname, `</em>`, ``)
			list = append(list, searchresult{
				Roomid:  j.Data.Result[i].Roomid,
				Uname:   uname,
				Is_live: j.Data.Result[i].IsLive,
			})
		}

	}

	return
}

func KeepConnect() {
	for !IsConnected() {
		time.Sleep(time.Duration(30) * time.Second)
	}
}

func IsConnected() bool {
	apilog := apilog.Base_add(`IsConnected`)

	v, ok := c.K_v.LoadV(`网络中断不退出`).(bool)
	if !ok || !v {
		return true
	}

	req := reqf.New()
	if err := req.Reqf(reqf.Rval{
		Url:              "https://www.bilibili.com",
		Proxy:            kv.LoadV(`Common.Proxy`).(string),
		Timeout:          10 * 1000,
		JustResponseCode: true,
	}); err != nil {
		apilog.L(`W: `, `网络中断`, err)
		return false
	}

	apilog.L(`T: `, `已连接`)
	return true
}
