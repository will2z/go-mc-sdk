package client

import (
	"context"
	"errors"
	"fmt"
	"github.com/filswan/go-swan-lib/client"
	"github.com/filswan/go-swan-lib/client/web"
	"github.com/filswan/go-swan-lib/logs"
	shell "github.com/ipfs/go-ipfs-api"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func pathJoin(root string, parts ...string) string {
	url := root

	for _, part := range parts {
		url = strings.TrimRight(url, "/") + "/" + strings.TrimLeft(part, "/")
	}
	url = strings.TrimRight(url, "/")

	return url
}

func downloadFileByAria2(conf *Aria2Conf, downUrl, outPath string) error {
	aria2 := client.GetAria2Client(conf.Host, conf.Secret, conf.Port)
	outDir := filepath.Dir(outPath)
	fileName := filepath.Base(outPath)
	logs.GetLogger().Infof("start download by aria2, downUrl:%s, outDir:%s, fileName:%s", downUrl, outDir, fileName)
	aria2Download := aria2.DownloadFile(downUrl, outDir, fileName)
	if aria2Download == nil {
		logs.GetLogger().Error("no response when asking aria2 to download")
		return errors.New("no response when asking aria2 to download")
	}

	if aria2Download.Error != nil {
		logs.GetLogger().Error(aria2Download.Error.Message)
		return errors.New(aria2Download.Error.Message)
	}

	if aria2Download.Gid == "" {
		logs.GetLogger().Error("no gid returned when asking aria2 to download")
		return errors.New("no gid returned when asking aria2 to download")
	}

	logs.GetLogger().Info("can check download status by gid:", aria2Download.Gid)
	return nil
}

func httpPost(uri, key, token string, params interface{}) ([]byte, error) {
	response, err := web.HttpRequestWithKey(http.MethodPost, uri, key, token, params)
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err
	}
	return response, nil
}

func isFile(dirFullPath string) (*bool, error) {
	fi, err := os.Stat(dirFullPath)

	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err
	}

	switch mode := fi.Mode(); {
	case mode.IsDir():
		isFile := false
		return &isFile, nil
	case mode.IsRegular():
		isFile := true
		return &isFile, nil
	default:
		err := fmt.Errorf("unknown path type")
		logs.GetLogger().Error(err)
		return nil, err
	}
}

func dirSize(path string) int64 {
	var size int64
	entrys, err := os.ReadDir(path)
	if err != nil {
		logs.GetLogger().Error(err)
		return 0
	}
	for _, entry := range entrys {
		if entry.IsDir() {
			size += dirSize(filepath.Join(path, entry.Name()))
		} else {
			info, err := entry.Info()
			if err == nil {
				size += info.Size()
			}
		}
	}
	return size
}

func walkDirSize(path string) int64 {
	var totalSize int64
	filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			fileInfo, err := os.Stat(path)
			if err == nil {
				fileSize := fileInfo.Size()
				totalSize += fileSize
			}
		}
		return nil
	})
	return totalSize
}

func uploadFileToIpfs(sh *shell.Shell, fileName string) (string, error) {

	file, err := os.Open(fileName)
	if err != nil {
		logs.GetLogger().Error(err)
		return "", err
	}
	defer file.Close()

	dataCid, err := sh.Add(file)
	if err != nil {
		logs.GetLogger().Error(err)
		return "", err
	}

	destPath := "/"
	srcPath := pathJoin("/ipfs/", dataCid)
	err = sh.FilesCp(context.Background(), srcPath, destPath)
	if err != nil {
		logs.GetLogger().Error(err)
		return "", err
	}

	return dataCid, nil
}

func uploadDirToIpfs(sh *shell.Shell, dirName string) (string, error) {

	dataCid, err := sh.AddDir(dirName)
	if err != nil {
		logs.GetLogger().Error(err)
		return "", err
	}

	destPath := "/"
	srcPath := pathJoin("/ipfs/", dataCid)
	err = sh.FilesCp(context.Background(), srcPath, destPath)
	if err != nil {
		logs.GetLogger().Error(err)
		return "", err
	}

	return dataCid, nil
}

func dataCidIsDir(sh *shell.Shell, dataCid string) (*bool, error) {

	path := pathJoin("/ipfs/", dataCid)
	stat, err := sh.FilesStat(context.Background(), path)
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err
	}
	logs.GetLogger().Debug("FileStat:", stat)

	isFile := false
	if stat.Type == "directory" {
		isFile = true
	}

	return &isFile, nil
}

func downloadFromIpfs(sh *shell.Shell, dataCid, outDir string) error {
	return sh.Get(dataCid, outDir)
}

func NewNode(hash, path, name string, size uint64, dir bool) *TreeNode {
	return &TreeNode{
		Hash:  hash,
		Path:  path,
		Name:  name,
		Size:  size,
		Dir:   dir,
		Child: []*TreeNode{},
	}
}

func NewNodeByDataCid(sh *shell.Shell, dataCid string, nodePath, name string) *TreeNode {
	path := pathJoin("/ipfs/", dataCid)
	stat, err := sh.FilesStat(context.Background(), path)
	if err != nil {
		logs.GetLogger().Error(dataCid, " get dag directory info err:", err)
		return nil
	}

	if stat.Type == "directory" {
		return NewNode(dataCid, pathJoin(nodePath, dataCid), name, stat.CumulativeSize, true)
	} else if stat.Type == "file" {
		return NewNode(dataCid, pathJoin(nodePath, dataCid), name, stat.CumulativeSize, false)
	} else {
		logs.GetLogger().Warn("unknown type in build node: ", stat.Type)
	}

	return nil
}

func (n *TreeNode) AddChild(node *TreeNode) error {

	if n.Child != nil {
		n.Child = append(n.Child, node)
	}

	return nil
}

func (n *TreeNode) BuildChildTree(sh *shell.Shell) error {
	if !n.Dir || len(n.Child) == 0 {
		return nil
	}

	for _, child := range n.Child {
		if !child.Dir {
			continue
		}

		resp := DagGetResponse{}
		if err := sh.DagGet(child.Hash, &resp); err != nil {
			logs.GetLogger().Error(child.Hash, " get dag directory info err:", err)
			continue
		}

		// build all subChild
		for _, link := range resp.Links {
			subChild := NewNodeByDataCid(sh, link.Hash.Target, child.Path, link.Name)
			if subChild == nil {
				continue
			}

			child.AddChild(subChild)
		}

		child.BuildChildTree(sh)

	}

	return nil
}

func (n *TreeNode) Insert(hash string, node *TreeNode) error {

	prev := n.Find(hash)
	if prev != nil {
		prev.Child = append(prev.Child, node)
	}

	return nil
}

func (n *TreeNode) Del(hash string) error {
	//TODO:
	return nil
}

func (n *TreeNode) Find(hash string) *TreeNode {
	if n.Hash == hash {
		return n
	}

	for _, child := range n.Child {
		if fn := child.Find(hash); fn != nil {
			return fn
		}
	}

	return nil
}

func (n *TreeNode) PrintAll() error {

	n.Print()

	for _, child := range n.Child {
		child.PrintAll()
	}

	return nil
}

func (n *TreeNode) Print() error {
	//logs.GetLogger().Infof("TreeNode: hash=%s, path=%s, name=%s, size=%d, deep=%d, dir=%t, child-num=%d",
	//	n.Hash, n.Path, n.Name, n.Size, n.Deep, n.Dir, len(n.Child))
	fmt.Print("\n")
	if n.Path == "/" {
		fmt.Printf("/")
		return nil
	}

	count := strings.Count(n.Path, "/")
	for i := 0; i < count-1; i++ {
		fmt.Print("    ")
	}
	fmt.Printf("|---%s (Hash:%s Size:%d)", n.Name, n.Hash, n.Size)

	return nil
}
