package cv

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/dustin/go-humanize"
	file "github.com/qydysky/part/file"
	idpool "github.com/qydysky/part/idpool"
	log "github.com/qydysky/part/log"
	mq "github.com/qydysky/part/msgq"
	reqf "github.com/qydysky/part/reqf"
	syncmap "github.com/qydysky/part/sync"
	web "github.com/qydysky/part/web"
)

type Common struct {
	PID               int                   //进程id
	Uid               int                   //client uid
	Live              []LiveQn              //直播流链接
	Live_qn           int                   //当前直播流质量
	Live_want_qn      int                   //期望直播流质量
	Roomid            int                   //房间ID
	Cookie            syncmap.Map           //Cookie
	Title             string                //直播标题
	Uname             string                //主播名
	UpUid             int                   //主播uid
	Rev               float64               //营收
	Renqi             int                   //人气
	Watched           int                   //观看人数
	OnlineNum         int                   //在线人数
	GuardNum          int                   //舰长数
	ParentAreaID      int                   //父分区
	AreaID            int                   //子分区
	Locked            bool                  //直播间封禁
	Note              string                //分区排行
	Live_Start_Time   time.Time             //直播开始时间
	Liveing           bool                  //是否在直播
	Wearing_FansMedal int                   //当前佩戴的粉丝牌
	Token             string                //弹幕钥
	WSURL             []string              //弹幕链接
	LIVE_BUVID        bool                  //cookies含LIVE_BUVID
	Stream_url        *url.URL              //直播Web服务
	Proxy             string                //全局代理
	AcceptQn          map[int]string        //允许的直播流质量
	Qn                map[int]string        //全部直播流质量
	StreamType        StreamType            //当前直播流类型
	AllStreamType     map[string]StreamType //直播流类型
	K_v               syncmap.Map           //配置文件
	Log               *log.Log_interface    //日志
	Danmu_Main_mq     *mq.Msgq              //消息
	ReqPool           *idpool.Idpool        //请求池
	SerF              *web.WebPath          //web服务处理
}

type LiveQn struct {
	Url          string
	ReUpTime     time.Time
	disableCount int
	Expires      int //流到期时间
}

func (t *LiveQn) Host() string {
	if liveUrl, e := url.Parse(t.Url); e == nil {
		return liveUrl.Host
	} else {
		panic(e)
	}
}

func (t *LiveQn) Valid() bool {
	return time.Now().After(t.ReUpTime)
}

// 自动停用机制
func (t *LiveQn) DisableAuto() {
	if time.Now().After(t.ReUpTime.Add(time.Minute).Add(time.Second * time.Duration(10*t.disableCount))) {
		t.disableCount = 0
	}
	t.disableCount += 1
	t.ReUpTime = time.Now().Add(time.Minute).Add(time.Second * time.Duration(10*t.disableCount))
}

func (t *LiveQn) Disable(reUpTime time.Time) {
	t.ReUpTime = reUpTime
}

// 自动停用机制
func (t *Common) DisableLiveAuto(host string) {
	for i := 0; i < len(t.Live); i++ {
		if liveUrl, e := url.Parse(t.Live[i].Url); e == nil {
			if host == liveUrl.Host {
				t.Live[i].DisableAuto()
				break
			}
		}
	}
}

func (t *Common) DisableLive(host string, reUpTime time.Time) {
	for i := 0; i < len(t.Live); i++ {
		if liveUrl, e := url.Parse(t.Live[i].Url); e == nil {
			if host == liveUrl.Host {
				t.Live[i].ReUpTime = reUpTime
				break
			}
		}
	}
}

func (t *Common) ValidLive() *LiveQn {
	for i := 0; i < len(t.Live); i++ {
		if time.Now().Before(t.Live[i].ReUpTime) {
			continue
		}
		return &t.Live[i]
	}
	return nil
}

type StreamType struct {
	Protocol_name string
	Format_name   string
	Codec_name    string
}

func (t *Common) Init() Common {
	t.PID = os.Getpid()

	t.AllStreamType = map[string]StreamType{
		`fmp4`: {
			Protocol_name: "http_hls",
			Format_name:   "fmp4",
			Codec_name:    "avc",
		},
		`flv`: {
			Protocol_name: "http_stream",
			Format_name:   "flv",
			Codec_name:    "avc",
		},
		`fmp4H`: {
			Protocol_name: "http_hls",
			Format_name:   "fmp4",
			Codec_name:    "hevc",
		},
		`flvH`: {
			Protocol_name: "http_stream",
			Format_name:   "flv",
			Codec_name:    "hevc",
		},
	}

	t.Qn = map[int]string{ // no change
		20000: "4K",
		10000: "原画",
		400:   "蓝光",
		250:   "超清",
		150:   "高清",
		80:    "流畅",
	}

	t.AcceptQn = map[int]string{ // no change
		20000: "4K",
		10000: "原画",
		400:   "蓝光",
		250:   "超清",
		150:   "高清",
		80:    "流畅",
	}

	t.Danmu_Main_mq = mq.New()

	t.ReqPool = idpool.New(func() interface{} {
		return reqf.New()
	})

	var (
		ckv     = flag.String("ckv", "", "自定义配置KV文件，将会覆盖config_K_v配置")
		roomIdP = flag.Int("r", 0, "roomid")
	)
	testing.Init()
	flag.Parse()
	t.Roomid = *roomIdP

	if e := t.loadConf(*ckv); e != nil {
		panic(e)
	}

	go func() {
		for {
			v, ok := t.K_v.LoadV("几秒后重载").(float64)
			if !ok || v < 0 {
				break
			} else if v < 60 {
				v = 60
			}
			time.Sleep(time.Duration(int(v)) * time.Second)

			if e := t.loadConf(*ckv); e != nil {
				fmt.Println(e)
			}
		}
	}()

	if serAdress, ok := t.K_v.LoadV("Web服务地址").(string); ok {
		serUrl, e := url.Parse("http://" + serAdress)
		if e != nil {
			panic(e)
		}

		t.SerF = new(web.WebPath)

		web.NewSyncMap(&http.Server{
			Addr: serUrl.Host,
		}, t.SerF)

		t.SerF.Store("/", func(w http.ResponseWriter, _ *http.Request) {
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)

			type s struct {
				MenInUse     string `json:"menInUse"`
				NumGoroutine int    `json:"numGoroutine"`
				GoVersion    string `json:"goVersion"`
			}
			type j struct {
				Stats s `json:"stats"`
			}

			ResStruct{0, "ok",
				j{
					s{
						humanize.Bytes(memStats.HeapInuse + memStats.StackInuse),
						runtime.NumGoroutine(),
						runtime.Version(),
					},
				},
			}.Write(w)
		})

		t.Stream_url, _ = url.Parse(`http://` + serAdress)
	}

	if val, exist := t.K_v.Load("http代理地址"); exist {
		t.Proxy = val.(string)
	}

	// 配置直播流类型
	if val, exist := t.K_v.Load("直播流类型"); exist {
		if st, ok := t.AllStreamType[val.(string)]; ok {
			t.StreamType = st
		} else {
			panic("未找到设定类型" + val.(string))
		}
	}

	{
		v, _ := t.K_v.LoadV("日志文件输出").(string)
		t.Log = log.New(log.Config{
			File:   v,
			Stdout: true,
			Prefix_string: map[string]struct{}{
				`T: `: log.On,
				`I: `: log.On,
				`N: `: log.On,
				`W: `: log.On,
				`E: `: log.On,
			},
		})
		logmap := make(map[string]struct{})
		if array, ok := t.K_v.Load(`日志显示`); ok {
			for _, v := range array.([]interface{}) {
				logmap[v.(string)] = log.On
			}
		}
		t.Log = t.Log.Level(logmap)
	}

	return *t
}

func (t *Common) loadConf(customConf string) error {
	var data map[string]interface{}

	// 64k
	if bb, e := file.New("config/config_K_v.json", 0, true).ReadAll(100, 1<<16); e != nil {
		if !errors.Is(e, io.EOF) {
			return e
		} else {
			json.Unmarshal(bb, &data)
		}
	}

	if customConf != "" {
		if strings.Contains(customConf, "http:") || strings.Contains(customConf, "https:") {
			//从网址读取
			req := t.ReqPool.Get()
			r := req.Item.(*reqf.Req)
			if e := r.Reqf(reqf.Rval{
				Url: customConf,
				Header: map[string]string{
					`User-Agent`:      `Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:103.0) Gecko/20100101 Firefox/103.0`,
					`Accept`:          `*/*`,
					`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
					`Accept-Encoding`: `gzip, deflate, br`,
					`Pragma`:          `no-cache`,
					`Cache-Control`:   `no-cache`,
					`Connection`:      `close`,
				},
				Timeout: 10 * 1000,
			}); e != nil {
				return errors.New("无法获取自定义配置文件 " + e.Error())
			}
			if r.Response == nil {
				return errors.New("无法获取自定义配置文件 响应为空")
			} else if r.Response.StatusCode&200 != 200 {
				return fmt.Errorf("无法获取自定义配置文件 %d", r.Response.StatusCode)
			} else {
				var tmp map[string]interface{}
				json.Unmarshal(r.Respon, &tmp)
				for k, v := range tmp {
					data[k] = v
				}
			}
			t.ReqPool.Put(req)
		} else {
			//从文件读取
			if bb, err := file.New(customConf, 0, true).ReadAll(100, 1<<16); err != nil {
				if errors.Is(err, io.EOF) {
					var tmp map[string]interface{}
					json.Unmarshal(bb, &tmp)
					for k, v := range tmp {
						data[k] = v
					}
				} else {
					return err
				}
			}
		}
	}

	for k, v := range data {
		t.K_v.Store(k, v)
	}

	return nil
}

var C = new(Common).Init()

// 消息队列
type Danmu_Main_mq_item struct {
	Class string
	Data  interface{}
}

// Web服务响应格式
type ResStruct struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func (t ResStruct) Write(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	data, e := json.Marshal(t)
	if e != nil {
		t.Code = -1
		t.Data = nil
		t.Message = e.Error()
		data, _ = json.Marshal(t)
	}
	w.Write(data)
}
