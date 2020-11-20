//+build gtk

package reply

import (
	"container/list"
	"errors"
	"strconv"
	"time"
	"strings"
	"log"
	"fmt"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/gdk"
	p "github.com/qydysky/part"
	F "github.com/qydysky/bili_danmu/F"
	c "github.com/qydysky/bili_danmu/CV"
	s "github.com/qydysky/part/buf"
)
const (
	max = 50
	max_keep = 5
	max_img = 500
)

const appId = "com.github.qydysky.bili_danmu.reply"

type gtk_list struct {
	text *gtk.TextView
	img *gtk.Image
	handle glib.SignalHandle
}
var pro_style *gtk.CssProvider
var gtkGetList = list.New()

var imgbuf = make(map[string](*gdk.Pixbuf))
var keep_list = list.New()

var keep_key = map[string]int{
	"face/0default":0,
	"face/0room":0,
	"face/0buyguide":9,
	"face/0gift":8,
	"face/0jiezou":8,
	"face/0level1":5,
	"face/0level2":3,
	"face/0level3":1,
	"face/0superchat":13,
}
var (
	Gtk_on bool
	Gtk_Tra bool
	Gtk_img_path string = "face"
	Gtk_danmuChan chan string = make(chan string, 1000)
	Gtk_danmuChan_uid chan string = make(chan string, 1000)
)

func init(){
	if!IsOn("Gtk") {return}
	go func(){
		{//加载特定信息驻留时长
			buf := s.New()
			buf.Load("config/config_gtk_keep_key.json")
			for k,v := range buf.B {
				keep_key[k] = int(v.(float64))
			}
		}
		go Gtk_danmu()
		var (
			sig = Danmu_mq.Sig()
			data interface{}
		)
		for {
			data,sig = Danmu_mq.Pull(sig)
			Gtk_danmuChan_uid <- data.(Danmu_mq_t).uid 
			Gtk_danmuChan <- data.(Danmu_mq_t).msg
		}
	}()
}

func Gtk_danmu() {
	if Gtk_on {return}
	gtk.Init(nil)

	var y func(string,string)
	var win *gtk.Window
	var scrolledwindow0 *gtk.ScrolledWindow
	var viewport0 *gtk.Viewport
	var w2_textView0 *gtk.TextView
	var w2_textView1 *gtk.TextView
	var w2_Entry0 *gtk.Entry
	var w2_Entry0_editting bool

	application, err := gtk.ApplicationNew(appId, glib.APPLICATION_FLAGS_NONE)
	if err != nil {log.Println(err);return}

	application.Connect("startup", func() {
		log.Println("application startup")	
		var grid0 *gtk.Grid;

		builder, err := gtk.BuilderNewFromFile("ui/1.glade")
		if err != nil {log.Println(err);return}
		builder2, err := gtk.BuilderNewFromFile("ui/2.glade")
		if err != nil {log.Println(err);return}

		{
			signals := map[string]interface{}{
				"on_main_window_destroy": onMainWindowDestroy,
			}
			builder.ConnectSignals(signals)
			builder2.ConnectSignals(signals)
		}

		{
			obj, err := builder.GetObject("main_window")
			if err != nil {log.Println(err);return}
			win, err = isWindow(obj)
			if err != nil {log.Println(err);return}
			application.AddWindow(win)
			defer win.ShowAll()
		}
		{
			obj, err := builder2.GetObject("main_window")
			if err != nil {log.Println(err);return}
			win2, err := isWindow(obj)
			if err != nil {log.Println(err);return}
			application.AddWindow(win2)
			defer win2.ShowAll()
		}
		{//营收
			obj, err := builder2.GetObject("t0")
			if err != nil {log.Println(err);return}
			if tmp,ok := obj.(*gtk.TextView); ok {
				w2_textView0 = tmp
			}else{log.Println("cant find #t0 in .glade");return}
		}
		{//直播时长
			obj, err := builder2.GetObject("t1")
			if err != nil {log.Println(err);return}
			if tmp,ok := obj.(*gtk.TextView); ok {
				w2_textView1 = tmp
			}else{log.Println("cant find #t1 in .glade");return}
		}
		{//发送弹幕
			var danmu_send_form string
			{//发送弹幕格式
				obj, err := builder2.GetObject("send_danmu_form")
				if err != nil {log.Println(err);return}
				if tmp,ok := obj.(*gtk.Entry); ok {
					tmp.Connect("focus-out-event", func() {
						if t,e := tmp.GetText();e == nil && t != ``{
							danmu_send_form = t
							log.Println("弹幕格式已设置为",danmu_send_form)
						}
					})
				}else{log.Println("cant find #send_danmu in .glade");return}
			}
			obj, err := builder2.GetObject("send_danmu")
			if err != nil {log.Println(err);return}
			if tmp,ok := obj.(*gtk.Entry); ok {
				tmp.Connect("key-release-event", func(entry *gtk.Entry, event *gdk.Event) {
					eventKey := gdk.EventKeyNewFromEvent(event)
					if eventKey.KeyVal() == gdk.KEY_Return {
						if t,e := entry.GetText();e == nil && t != ``{
							danmu_want_send := t
							if danmu_send_form != `` {danmu_want_send = strings.ReplaceAll(danmu_send_form, "{D}", t)}
							if len([]rune(danmu_want_send)) > 20 {
								log.Println(`弹幕长度大于20,不做格式处理`)
								danmu_want_send = t
							} 
							Msg_senddanmu(danmu_want_send)
							entry.SetText(``)
						}
					}
				})
			}else{log.Println("cant find #send_danmu in .glade");return}
		}
		{//房间id
			obj, err := builder2.GetObject("want_room_id")
			if err != nil {log.Println(err);return}
			if tmp,ok := obj.(*gtk.Entry); ok {
				w2_Entry0 = tmp
				tmp.Connect("focus-in-event", func() {
					w2_Entry0_editting = true
				})
				tmp.Connect("focus-out-event", func() {
					w2_Entry0_editting = false
				})
			}else{log.Println("cant find #want_room_id in .glade");return}
		}
		{//房间id click
			obj, err := builder2.GetObject("want_click")
			if err != nil {log.Println(err);return}
			if tmp,ok := obj.(*gtk.Button); ok {
				tmp.Connect("clicked", func() {
					if t,e := w2_Entry0.GetText();e != nil {
						y("读取错误",load_face("0room"))
					} else if t != `` {
						if i,e := strconv.Atoi(t);e != nil {
							y(`输入错误`,load_face("0room"))
						} else {
							c.Roomid =  i
							c.Danmu_Main_mq.Push(c.Danmu_Main_mq_item{
								Class:`change_room`,
							})
						}
					} else {
						y(`房间号输入为空`,load_face("0room"))
					}
				})
			}else{log.Println("cant find #want_click in .glade");return}
		}
		{
			obj, err := builder.GetObject("scrolledwindow0")
			if err != nil {log.Println(err);return}
			if tmp,ok := obj.(*gtk.ScrolledWindow); ok {
				scrolledwindow0 = tmp
			}else{log.Println("cant find #scrolledwindow0 in .glade");return}
		}

		{
			obj, err := builder.GetObject("viewport0")
			if err != nil {log.Println(err);return}
			if tmp,ok := obj.(*gtk.Viewport); ok {
				viewport0 = tmp
			}else{log.Println("cant find #viewport0 in .glade");return}
		}

		{
			obj, err := builder.GetObject("grid0")
			if err != nil {log.Println(err);return}
			if tmp,ok := obj.(*gtk.Grid); ok {
				grid0 = tmp
			}else{log.Println("cant find #grid0 in .glade");return}
		}

		imgbuf["face/0default"],_ = gdk.PixbufNewFromFileAtSize("face/0default", 40, 40);

		{
			var e error
			if pro_style,e = gtk.CssProviderNew();e == nil{
				if e = pro_style.LoadFromPath(`ui/1.css`);e == nil{
					if scr := win.GetScreen();scr != nil {
						gtk.AddProviderForScreen(scr,pro_style,gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
					}
				}else{log.Println(e)}
			}else{log.Println(e)}
		}

		y = func(s,img_src string){
			var tmp_list gtk_list

			tmp_list.text,_ = gtk.TextViewNew();
			{
				tmp_list.text.SetMarginStart(5)
				tmp_list.text.SetEditable(false)
				tmp_list.text.SetHExpand(true)
				tmp_list.text.SetWrapMode(gtk.WRAP_WORD_CHAR)
			}
			{
				var e error
				tmp_list.handle,e = tmp_list.text.Connect("size-allocate", func(){

					b,e := tmp_list.text.GetBuffer()
					if e != nil {log.Println(e);return}
					b.SetText(s)

					{
						var e error
						tmp := scrolledwindow0.GetVAdjustment()
						h := viewport0.GetViewWindow().WindowGetHeight()
						if tmp.GetUpper() - tmp.GetValue() < float64(h) * 1.7 {
							tmp.SetValue(tmp.GetUpper() - float64(h))
						}
						if e != nil {log.Println(e)}
					}
				})
				if e != nil {log.Println(e)}
			}

			tmp_list.img,_ =gtk.ImageNew();
			{
				var (
					pixbuf *gdk.Pixbuf
					e error
				)
				if v,ok := imgbuf[img_src];ok{
					pixbuf,e = gdk.PixbufCopy(v)
				} else {
					pixbuf,e = gdk.PixbufNewFromFileAtSize(img_src, 40, 40);
					if e == nil {
						if len(imgbuf) > max_img {
							for k,_ := range imgbuf {delete(imgbuf,k);break}
						}
						imgbuf[img_src],e = gdk.PixbufCopy(pixbuf)
					}
				}
				if e == nil {tmp_list.img.SetFromPixbuf(pixbuf)}
			}
			{
				loc := int(grid0.Container.GetChildren().Length())/2;
				sec := 0
				if tsec,ok := keep_key[img_src];ok && tsec != 0 {
					sec = tsec
					if sty,e := tmp_list.text.GetStyleContext();e == nil{
						sty.AddClass("highlight")
					}
				}
				/*
					front
					|
					back index:0
				*/
				var InsertIndex int = keep_list.Len()
				if sec > InsertIndex / max_keep {
					var cu_To = time.Now().Add(time.Second * time.Duration(sec))
					var hasInsert bool
					for el := keep_list.Front(); el != nil; el = el.Next(){
						if cu_To.After(el.Value.(time.Time)) {InsertIndex -= 1;continue}
						keep_list.InsertBefore(cu_To,el)
						hasInsert = true
						break
					}
					if !hasInsert {
						keep_list.PushBack(cu_To)
					}
				}
				grid0.InsertRow(loc - InsertIndex);
				grid0.Attach(tmp_list.img, 0, loc - InsertIndex, 1, 1)
				grid0.Attach(tmp_list.text, 1, loc - InsertIndex, 1, 1)

				loc = int(grid0.Container.GetChildren().Length())/2;
				for loc > max {
					if i,e := grid0.GetChildAt(0,0); e != nil{i.(*gtk.Widget).Destroy()}
					if i,e := grid0.GetChildAt(1,0); e != nil{i.(*gtk.Widget).Destroy()}
					grid0.RemoveRow(0)
					loc -= 1
				}
			}

			win.ShowAll()
		}


		Gtk_on = true
	})

	application.Connect("activate", func() {
		log.Println("application activate")
		go func(){
			for{
				time.Sleep(time.Second)
				if len(Gtk_danmuChan) == 0 {continue}
				for el := keep_list.Front(); el != nil && time.Now().After(el.Value.(time.Time));el = el.Next() {
					keep_list.Remove(el)
				}
				glib.TimeoutAdd(uint(1000 / (len(Gtk_danmuChan) + 1)),func()(bool){
					if len(Gtk_danmuChan) == 0 {return false}
					y(<-Gtk_danmuChan,load_face(<-Gtk_danmuChan_uid))
					return true
				})
			}
		}()

		glib.TimeoutAdd(uint(3000), func()(o bool){
			o = true
			//y("sssss",load_face(""))
			{//营收
				if IsOn("ShowRev") {
					b,e := w2_textView0.GetBuffer()
					if e != nil {log.Println(e);return}
					b.SetText(fmt.Sprintf("￥%.2f",c.Rev))					
				}
			}
			{//时长
				if c.Liveing {
					b,e := w2_textView1.GetBuffer()
					if e != nil {log.Println(e);return}
					d := time.Since(c.Live_Start_Time).Round(time.Second)
					h := d / time.Hour
					d -= h * time.Hour
					m := d / time.Minute
					d -= m * time.Minute
					s := d / time.Second
					b.SetText(fmt.Sprintf("%02d:%02d:%02d", h, m, s))					
				}
			}
			{//房间id
				if !w2_Entry0_editting {
					if t,e := w2_Entry0.GetText();e == nil && t == `` && c.Roomid != 0{
						w2_Entry0.SetText(strconv.Itoa(c.Roomid))
					}
				}
			}
			if gtkGetList.Len() == 0 {return}
			el := gtkGetList.Front()
			if el == nil {return}
			if uid,ok := gtkGetList.Remove(el).(string);ok{
				go func(){
					src := F.Get_face_src(uid)
					if src == "" {return}
					req := p.Req()
					if e := req.Reqf(p.Rval{
						Url:src,
						SaveToPath:Gtk_img_path + `/` + uid,
						Timeout:3,
					}); e != nil{log.Println(e);}
				}()
			}

			return
		})
	})

	application.Connect("shutdown", func() {
		log.Println("application shutdown")	
		Gtk_on = false
	})

	application.Run(nil)
}

func isWindow(obj glib.IObject) (*gtk.Window, error) {
	if win, ok := obj.(*gtk.Window); ok {
		return win, nil
	}
	return nil, errors.New("not a *gtk.Window")
}

func onMainWindowDestroy() {
	log.Println("onMainWindowDestroy")
}

func load_face(uid string) (loc string) {
	loc = Gtk_img_path + `/` + "0default"
	if uid == "" {return}
	if _,ok := keep_key[Gtk_img_path + `/` + uid];ok{
		loc = Gtk_img_path + `/` + uid
		return
	}
	if p.Checkfile().IsExist(Gtk_img_path + `/` + uid) && p.Rand().MixRandom(1,100) > 1 {
		loc = Gtk_img_path + `/` + uid
		return
	}
	if gtkGetList.Len() > 1000 {return}
	gtkGetList.PushBack(uid)
	return
}