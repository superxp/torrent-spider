package main

import (
	"fmt"
	"github.com/bttown/bloomfilter"
	"github.com/bttown/dht"
	"github.com/bttown/metadata"
	"log"
	"os"
)

var hashFilter = bloomfilter.New(10000000)
var blackList = bloomfilter.New(10000000)

func saveTorrentFile(name string, metadata *metadata.Metadata) {
	f, err := os.Create(name)
	if err != nil {
		return
	}
	defer f.Close()

	f.Write(metadata.Torrent())
}

func main() {
	c := metadata.NewCollector()
	defer c.Close()

	c.OnFinish(func(req *metadata.Request, meta *metadata.Metadata) {
		// 过滤掉重复资源
		if hashFilter.MightContains(req.HashInfo) {
			return
		}
		hashFilter.Put(req.HashInfo)
		magnetLink := fmt.Sprintf("magnet:?xt=urn:btih:%s", req.HashInfo)
		torrentFileName := fmt.Sprintf("torrents/%s.torrent", meta.Name)
		log.Println("[Metadata]", magnetLink, meta.Name)
		saveTorrentFile(torrentFileName, meta)
	})

	c.OnError(func(req *metadata.Request, err error) {
		// 将无法访问的节点地址加入黑名单
		blackList.Put(req.RemoteAddr())
		log.Println("[Error]", err)
	})

	node := dht.NewNode(dht.OptionAddress("0.0.0.0:8662"))
	node.PeerHandler = func(ip string, port int, hashInfo, peerID string) {
		// 过滤掉无法访问的节点
		if blackList.MightContains(fmt.Sprintf("%s:%d", ip, port)) {
			return
		}

		err := c.Get(&metadata.Request{
			IP:       ip,
			Port:     port,
			HashInfo: hashInfo,
			PeerID:   peerID,
		})
		if err != nil {
			panic(err)
		}

	}
	node.Serve()
}
