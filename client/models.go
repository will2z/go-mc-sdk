package client

type ClientConf struct {
	Key            string    `toml:"key"`
	Token          string    `toml:"token"`
	IpfsApiUrl     string    `toml:"ipfs_api_url"`
	IpfsGatewayUrl string    `toml:"ipfs_gateway_url"`
	MetaServerUrl  string    `toml:"meta_server_url"`
	Aria2          Aria2Conf `toml:"aria2"`
}

type Aria2Conf struct {
	Host   string `toml:"host"`
	Port   int    `toml:"port"`
	Secret string `toml:"secret"`
}

type JsonRpcParams struct {
	JsonRpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Id      int           `json:"id"`
}

// StoreSourceFile
type StoreSourceFileReq struct {
	SourceName  string `json:"source_name"`
	IsDirector  bool   `json:"is_director"`
	SourceSize  int64  `json:"source_size"`
	DataCid     string `json:"data_cid"`
	DownloadUrl string `json:"download_url"`
}

type StoreSourceFileResponse struct {
	JsonRpc string `json:"jsonrpc"`
	Result  struct {
		Code    string `json:"code"`
		Message string `json:"message,omitempty"`
	} `json:"result"`
	Id int `json:"id"`
}

//GetSourceFiles

type SourceFilePageReq struct {
	PageNum   int  `json:"page_num"`
	Size      int  `json:"size"`
	ShowStore bool `json:"show_store"`
}

type SourceFilePageResponse struct {
	JsonRpc string `json:"jsonrpc"`
	Result  struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Total     int64         `json:"total"`
			PageCount int64         `json:"pageCount"`
			Sources   []*SourceFile `json:"files"`
		} `json:"data"`
	} `json:"result"`
	Id int `json:"id"`
}

type SourceFile struct {
	SourceName  string       `json:"source_name"`
	DataCid     string       `json:"data_cid"`
	DownloadUrl string       `json:"download_url"`
	StorageList []*SplitFile `json:"storage_list"`
	SourceSize  int64        `json:"source_size"`
	IsDirector  bool         `json:"is_director"`
}

//GetDataCidByName

type DataCidResponse struct {
	JsonRpc string `json:"jsonrpc"`
	Result  struct {
		Code    string   `json:"code"`
		Message string   `json:"message,omitempty"`
		Data    []string `json:"data,omitempty"`
	} `json:"result"`
	Id int `json:"id"`
}

// GetSourceFileByDataCid

type SourceFileResponse struct {
	JsonRpc string `json:"jsonrpc"`
	Result  struct {
		Code    string     `json:"code"`
		Message string     `json:"message,omitempty"`
		Data    SourceFile `json:"data,omitempty"`
	} `json:"result"`
	Id int `json:"id"`
}

type SplitFile struct {
	FileName         string            `json:"file_name"`
	DataCid          string            `json:"data_cid"`
	FileSize         int64             `json:"file_size"`
	StorageProviders []StorageProvider `json:"storage_providers"`
}

type StorageProvider struct {
	StorageProviderId string `json:"storage_provider_id"`
	StorageStatus     string `json:"storage_status"`
	DealId            int64  `json:"deal_id"`
	DealCid           string `json:"deal_cid"` // proposal cid or uuid
}

// GetDownloadFileInfoByDataCid

type DownloadFileInfoResponse struct {
	JsonRpc string `json:"jsonrpc"`
	Result  struct {
		Code    string             `json:"code"`
		Message string             `json:"message,omitempty"`
		Data    []DownloadFileInfo `json:"data,omitempty"`
	} `json:"result"`
	Id int `json:"id"`
}

type DownloadFileInfo struct {
	SourceName  string `json:"source_name"`
	DownloadUrl string `json:"download_url"`
	IsDirector  bool   `json:"is_director"`
}

type DagLink struct {
	Hash struct {
		Target string `json:"/"`
	} `json:"Hash"`
	Name  string `json:"Name"`
	Tsize int64  `json:"Tsize"`
}
type DagGetResponse struct {
	Data struct {
		Target struct {
			Bytes string `json:"bytes"`
		} `json:"/"`
	} `json:"Data"`
	Links []DagLink `json:"Links,omitempty"`
}

type TreeNode struct {
	Path  string
	Name  string
	Hash  string
	Size  uint64
	Dir   bool
	Deep  int
	Child []*TreeNode
}

// list option
type listOption struct {
	ShowStorage bool
}

type ListOption interface {
	apply(*listOption)
}

type funcOption struct {
	f func(*listOption)
}

func (fdo *funcOption) apply(do *listOption) {
	fdo.f(do)
}

func showStorageOption(f func(*listOption)) *funcOption {
	return &funcOption{
		f: f,
	}
}

func WithShowStorage(show bool) ListOption {
	return showStorageOption(func(o *listOption) {
		o.ShowStorage = show
	})
}
func defaultOptions() listOption {
	return listOption{
		ShowStorage: false,
	}
}
