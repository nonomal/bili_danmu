package f

import (
	"github.com/qydysky/bili_danmu/Reply/F/danmuXml"
	"github.com/qydysky/bili_danmu/Reply/F/liveOver"
	"github.com/qydysky/bili_danmu/Reply/F/reSetMp4TimeStamp"
	comp "github.com/qydysky/part/component"
)

func init() {
	var linkMap = map[string][]string{
		"github.com/qydysky/bili_danmu/Reply.startRecDanmu.stop": {
			comp.Sign[danmuXml.Sign](`toXml`),
			comp.Sign[reSetMp4TimeStamp.Sign](`resetTS`),
			// comp.Sign[fmp4Tomp4.Sign](`conver`),
		},
		"github.com/qydysky/bili_danmu/Reply.SerF.player.ws": {
			comp.Sign[danmuXml.Sign](`toXml`),
		},
		"github.com/qydysky/bili_danmu/Reply.SerF.player.xml": {
			comp.Sign[danmuXml.Sign](`toXml`),
		},
		"github.com/qydysky/bili_danmu/Reply.preparing": {
			comp.Sign[liveOver.Sign](`sumup`),
		},
	}
	if e := comp.Link(linkMap); e != nil {
		panic(e)
	}
}
