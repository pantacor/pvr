package pvrapi

type PvrRemote struct {
	RemoteSpec         string   `json:"pvr-spec"`         // the pvr remote protocol spec available
	JsonGetUrl         string   `json:"json-get-url"`     // where to pvr post stuff
	JsonKey            string   `json:"json-key"`         // what key is to use in post json [default: json]
	ObjectsEndpointUrl string   `json:"objects-endpoint"` // where to store/retrieve objects
	PostUrl            string   `json:"post-url"`         // where to post/announce new revisions
	PostFields         []string `json:"post-fields"`      // what fields require input
	PostFieldsOpt      []string `json:"post-fields-opt"`  // what optional fields are available [default: <empty>]
}

type PvrRemoteObject struct {
	Id           string `json:"id" bson:"id"`
	StorageId    string `json:"storage-id" bson:"_id"`
	Owner        string `json:"owner"`
	ObjectName   string `json:"objectname"`
	Sha          string `json:"sha256sum"`
	Size         string `json:"size"`
	MimeType     string `json:"mime-type"`
	SignedPutUrl string `json:"signed-puturl"`
	SignedGetUrl string `json:"signed-geturl"`
	Now          string `json:"now"`
	ExpireTime   string `json:"expire-time"`
}
