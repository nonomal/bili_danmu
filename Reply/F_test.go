package reply

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	c "github.com/qydysky/bili_danmu/CV"
	file "github.com/qydysky/part/file"
	psql "github.com/qydysky/part/sql"
)

func TestSaveDanmuToDB(t *testing.T) {
	c.C.K_v.Store(`保存弹幕至db`, map[string]any{
		"dbname": "sqlite",
		"url":    "danmu.sqlite3",
		"create": "create table danmu (created text, createdunix text, msg text, color text, auth text, uid text, roomid text)",
		"insert": "insert into danmu  values ({Date},{Unix},{Msg},{Color},{Auth},{Uid},{Roomid})",
	})
	saveDanmuToDB.init(c.C)
	saveDanmuToDB.danmu(Danmu_item{
		msg:    "可能走位配合了他的压枪",
		color:  "#54eed8",
		auth:   "畏未",
		uid:    "96767379",
		roomid: 92613,
	})
	saveDanmuToDB.db.Close()

	if db, e := sql.Open("sqlite", "danmu.sqlite3"); e != nil {
		t.Fatal(e)
	} else {
		tx := psql.BeginTx[any](db, context.Background())
		tx.Do(psql.SqlFunc[any]{Query: "select msg as Msg from danmu"})
		tx.AfterQF(func(_ *any, rows *sql.Rows, txE error) (_ *any, stopErr error) {
			type row struct {
				Msg string
			}

			v, err := psql.DealRows(rows, func() row { return row{} })
			if err != nil {
				return nil, err
			}
			if len(*v) != 1 || (*v)[0].Msg != "可能走位配合了他的压枪" {
				return nil, errors.New("no msg")
			}
			return nil, nil
		})
		if _, e := tx.Fin(); e != nil {
			t.Fatal(e)
		}
		db.Close()
	}

	if e := file.New("danmu.sqlite3", 0, true).Delete(); e != nil {
		t.Fatal(e)
	}
}
