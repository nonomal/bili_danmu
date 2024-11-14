package f

import (
	"context"
	"net/http"

	_ "github.com/qydysky/bili_danmu/Reply/F/danmuCountPerMin"
	comp "github.com/qydysky/part/component2"
)

var DanmuCountPerMin = comp.Get[interface {
	// will WriteHeader
	GetRec(savePath string, r *http.Request, w http.ResponseWriter) error
	Rec(ctx context.Context, roomid int, savePath string)
	Do(roomid int)
}](`danmuCountPerMin`)
