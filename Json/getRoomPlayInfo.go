package part

type GetRoomPlayInfo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	TTL     int    `json:"ttl"`
	Data    struct {
		RoomID          int   `json:"room_id"`
		ShortID         int   `json:"short_id"`
		UID             int   `json:"uid"`
		IsHidden        bool  `json:"is_hidden"`
		IsLocked        bool  `json:"is_locked"`
		IsPortrait      bool  `json:"is_portrait"`
		LiveStatus      int   `json:"live_status"`
		HiddenTill      int   `json:"hidden_till"`
		LockTill        int   `json:"lock_till"`
		Encrypted       bool  `json:"encrypted"`
		PwdVerified     bool  `json:"pwd_verified"`
		LiveTime        int   `json:"live_time"`
		RoomShield      int   `json:"room_shield"`
		AllSpecialTypes []int `json:"all_special_types"`
		PlayurlInfo     struct {
			ConfJSON string `json:"conf_json"`
			Playurl  struct {
				Cid     int `json:"cid"`
				GQnDesc []struct {
					Qn       int         `json:"qn"`
					Desc     string      `json:"desc"`
					HdrDesc  string      `json:"hdr_desc"`
					AttrDesc interface{} `json:"attr_desc"`
				} `json:"g_qn_desc"`
				Stream []struct {
					ProtocolName string `json:"protocol_name"`
					Format       []struct {
						FormatName string `json:"format_name"`
						Codec      []struct {
							CodecName string `json:"codec_name"`
							CurrentQn int    `json:"current_qn"`
							AcceptQn  []int  `json:"accept_qn"`
							BaseURL   string `json:"base_url"`
							URLInfo   []struct {
								Host      string `json:"host"`
								Extra     string `json:"extra"`
								StreamTTL int    `json:"stream_ttl"`
							} `json:"url_info"`
							HdrQn     interface{} `json:"hdr_qn"`
							DolbyType int         `json:"dolby_type"`
							AttrName  string      `json:"attr_name"`
						} `json:"codec"`
					} `json:"format"`
				} `json:"stream"`
				P2PData struct {
					P2P      bool        `json:"p2p"`
					P2PType  int         `json:"p2p_type"`
					MP2P     bool        `json:"m_p2p"`
					MServers interface{} `json:"m_servers"`
				} `json:"p2p_data"`
				DolbyQn interface{} `json:"dolby_qn"`
			} `json:"playurl"`
		} `json:"playurl_info"`
	} `json:"data"`
}
