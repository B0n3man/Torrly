package main

import (
	"github.com/AcidOP/torrly/torrent"
)

func main() {
	t1, err := torrent.NewTorrentFromFile("./debian.torrent")
	if err != nil {
		panic(err)
	}

	t1.ViewTorrent()
	// t1.FirstTwentyHashes()
	t1.StartDownload()

	// t2, err := torrent.NewTorrentFromFile("./test.torrent")
	// if err != nil {
	// 	panic(err)
	// }

	// t2.ViewTorrent()
}
