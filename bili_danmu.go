package bili_danmu

import (
	"fmt"
	"flag"
	"time"
	"strconv"
	"os"
	"os/signal"

	p "github.com/qydysky/part"
	ws "github.com/qydysky/part/websocket"
	reply "github.com/qydysky/bili_danmu/Reply"
	c "github.com/qydysky/bili_danmu/CV"
	F "github.com/qydysky/bili_danmu/F"
)

var danmulog = p.Logf().New().Open("danmu.log").Base(-1, "bili_danmu.go").Level(c.LogLevel)

func Demo(roomid ...int) {

	danmulog.Base(-1, "测试")
	defer danmulog.Base(0)
	
	//ctrl+c退出
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)


	{
		var groomid = flag.Int("r", 0, "roomid")
		var live_qn = flag.String("q", "0", "qn")
		flag.Parse()
	
		if _,ok := c.Default_qn[*live_qn]; ok{c.Live_qn = *live_qn}

		var exit_sign bool
		var change_room_chan = make(chan bool,1)

		go func(){
			var room = *groomid
			if room == 0 && len(roomid) != 0 {
				room = roomid[0]
			}
			if room == 0 {
				fmt.Printf("输入房间号: ")
				_, err := fmt.Scanln(&room)
				if err != nil {
					danmulog.E("输入错误", err)
					return
				}
			} else {
				fmt.Print("房间号: ", strconv.Itoa(room), "\n")
			}
			if c.Roomid == 0 {
				c.Roomid = room
				change_room_chan <- true
			}
		}()

		go func(){
			var (
				sig = c.Danmu_Main_mq.Sig()
				data interface{}
			)
			for {
				data,sig = c.Danmu_Main_mq.Pull(sig)
				if d,ok := data.(c.Danmu_Main_mq_item);!ok {
					continue
				} else {
					switch d.Class {
					case `change_room`:
						c.Rev = 0.0 //营收
						c.Renqi = 1//人气置1
						c.GuardNum = 0//舰长数
						c.Note = ``//分区排行
						c.Uname = ``//主播id
						c.Title = ``
						reply.Saveflv_wait()//停止保存直播流
						change_room_chan <- true
					case `c.Rev_add`:
						c.Rev += d.Data.(float64)
					case `c.Renqi`:
						if tmp,ok := d.Data.(int);ok{
							c.Renqi = tmp
						}
					default:
					}
				}
			}
		}()

		<-change_room_chan

		for !exit_sign {
			//获取房间相关信息
			api := F.New_api(c.Roomid).Get_host_Token().Get_live()
			c.Roomid = api.Roomid

			//获取cookies
			{
				var q = p.Filel{
					Write:false,
				}
				if p.Checkfile().IsExist("cookie.txt") {
					q.File = "cookie.txt"
					f := p.File().FileWR(q)
					c.Cookie = f
				} else {
					danmulog.I("未检测到cookie.txt，如果需要登录请在本机打开以下网址扫码登录，不需要请忽略")
					go func(){//获取cookie
						api.Get_cookie()
						if c.Cookie != `` {
							danmulog.I("你已登录，刷新房间！")
							c.Danmu_Main_mq.Push(c.Danmu_Main_mq_item{//刷新
								Class:`change_room`,
							})
						}
					}()
					p.Sys().Timeoutf(3)
				}
			}
			//获取用户版本
			api.Get_Version()
			//切换粉丝牌，只在cookie存在时启用
			api.Switch_FansMedal()
			if len(api.Url) == 0 || api.Roomid == 0 || api.Token == "" || api.Uid == 0 || api.Locked {
				danmulog.E("some err")
				return
			}
			danmulog.I("连接到房间", c.Roomid)

			//对每个弹幕服务器尝试
			for _, v := range api.Url {
				//ws启动
				ws_c := ws.New_client(ws.Client{
					Url:v,
					TO:35 * 1000,
					Func_abort_close:func(){danmulog.I(`服务器连接中断`)},
					Func_normal_close:func(){danmulog.I(`服务器连接关闭`)},
					Header:map[string][]string{
						"Cookie":[]string{c.Cookie},
					},
				}).Handle()
				if ws_c.Isclose() {
					danmulog.E("连接错误", ws_c.Error())
					continue
				}

				//SendChan 传入发送[]byte
				//RecvChan 接收[]byte
				danmulog.I("连接", v)
				ws_c.SendChan <- F.HelloGen(c.Roomid, api.Token)
				if F.HelloChe(<- ws_c.RecvChan) {
					danmulog.I("已连接到房间", c.Uname, `(`, c.Roomid, `)`)
					reply.Gui_show(`进入直播间: `+c.Uname+` (`+strconv.Itoa(c.Roomid)+`)`, `0room`)
					if c.Title != `` {
						danmulog.I(c.Title)
						reply.Gui_show(`房间标题: `+c.Title, `0room`)
					}
					//30s获取一次人气
					go func(){
						p.Sys().MTimeoutf(500)//500ms
						danmulog.I("获取人气")
						go func(){
							heartbeatmsg, heartinterval := F.Heartbeat()
							for !ws_c.Isclose() {
								ws_c.SendChan <- heartbeatmsg
								time.Sleep(time.Millisecond*time.Duration(heartinterval))
							}
						}()

						//传输变量，以便响应弹幕"弹幕机在么"
						c.Live = api.Live
						//获取过往营收 舰长数量
						// go api.Get_OnlineGoldRank()//高能榜显示的是在线观众的打赏

						go func(){//订阅消息，以便刷新舰长数
							api.Get_guardNum()
							var (
								sig = c.Danmu_Main_mq.Sig()
								data interface{}
							)
							for {
								data,sig = c.Danmu_Main_mq.Pull(sig)
								if d,ok := data.(c.Danmu_Main_mq_item);!ok {
									continue
								} else {
									switch d.Class {
									case `guard_update`:
										go api.Get_guardNum()
									case `change_room`:
										return//换房时退出当前房间
									default:
									}
								}
							}
						}()

						if c.Cookie != `` {//附加功能 弹幕机 无cookie无法发送弹幕
							reply.Danmuji_auto(1)
						}
						{//附加功能 直播流保存 营收
							go reply.Saveflvf()
							go reply.ShowRevf()
						}
					}()
				}

				var isclose bool
				var break_sign bool
				for !isclose {
					select {
					case i := <- ws_c.RecvChan:
						if len(i) == 0 && ws_c.Isclose() {
							isclose = true
						} else {
							go reply.Reply(i)
						}
					case <- interrupt:
						ws_c.Close()
						danmulog.I("停止，等待服务器断开连接")
						break_sign = true
						exit_sign = true
					case <- change_room_chan:
						ws_c.Close()
						danmulog.I("停止，等待服务器断开连接")
						break_sign = true
					}

				}

				if break_sign {break}
			}

			p.Sys().Timeoutf(1)
		}

		close(interrupt)
		{//附加功能 直播流
			reply.Saveflv_wait()
		}
		danmulog.I("结束退出")
	}
}

