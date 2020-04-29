package torrent

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/anacrolix/envpprof"
	"github.com/anacrolix/missinggo"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	log "github.com/schollz/logger"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/xerrors"
)

func torrentBar(t *torrent.Torrent) {
	go func() {
		if t.Info() == nil {
			fmt.Printf("getting info for %q\n", t.Name())
			<-t.GotInfo()
		}
		bar := progressbar.NewOptions64(t.Length(),
			progressbar.OptionShowBytes(true),
		)
		previousCompleted := int64(0)
		for {
			var completedPieces, partialPieces int
			psrs := t.PieceStateRuns()
			for _, r := range psrs {
				if r.Complete {
					completedPieces += r.Length
				}
				if r.Partial {
					partialPieces += r.Length
				}
			}
			completedBytes := t.BytesCompleted()
			bar.Add64(completedBytes - previousCompleted)
			previousCompleted -= completedBytes
			// fmt.Printf(
			// 	"downloading %q: %s/%s, %d/%d pieces completed (%d partial)\n",
			// 	t.Name(),
			// 	humanize.Bytes(uint64(t.BytesCompleted())),
			// 	humanize.Bytes(uint64(t.Length())),
			// 	completedPieces,
			// 	t.NumPieces(),
			// 	partialPieces,
			// )
			time.Sleep(time.Second)
		}
	}()
}

func addTorrents(client *torrent.Client, arg string) error {
	t, err := func() (*torrent.Torrent, error) {
		if strings.HasPrefix(arg, "magnet:") {
			t, err := client.AddMagnet(arg)
			if err != nil {
				return nil, xerrors.Errorf("error adding magnet: %w", err)
			}
			return t, nil
		} else if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
			response, err := http.Get(arg)
			if err != nil {
				return nil, xerrors.Errorf("Error downloading torrent file: %s", err)
			}

			metaInfo, err := metainfo.Load(response.Body)
			defer response.Body.Close()
			if err != nil {
				return nil, xerrors.Errorf("error loading torrent file %q: %s\n", arg, err)
			}
			t, err := client.AddTorrent(metaInfo)
			if err != nil {
				return nil, xerrors.Errorf("adding torrent: %w", err)
			}
			return t, nil
		} else if strings.HasPrefix(arg, "infohash:") {
			t, _ := client.AddTorrentInfoHash(metainfo.NewHashFromHex(strings.TrimPrefix(arg, "infohash:")))
			return t, nil
		} else {
			metaInfo, err := metainfo.LoadFromFile(arg)
			if err != nil {
				return nil, xerrors.Errorf("error loading torrent file %q: %s\n", arg, err)
			}
			t, err := client.AddTorrent(metaInfo)
			if err != nil {
				return nil, xerrors.Errorf("adding torrent: %w", err)
			}
			return t, nil
		}
	}()
	if err != nil {
		return xerrors.Errorf("adding torrent for %q: %w", arg, err)
	}
	torrentBar(t)
	go func() {
		<-t.GotInfo()
		t.DownloadAll()
	}()
	return nil
}

func stdoutAndStderrAreSameFile() bool {
	fi1, _ := os.Stdout.Stat()
	fi2, _ := os.Stderr.Stat()
	return os.SameFile(fi1, fi2)
}

func exitSignalHandlers(notify *missinggo.SynchronizedEvent) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	for {
		log.Debugf("close signal received: %+v", <-c)
		notify.Set()
	}
}

func Download(torrentname string) error {
	return downloadErr(torrentname)
}

func downloadErr(torrentname string) error {
	defer envpprof.Stop()
	clientConfig := torrent.NewDefaultClientConfig()
	clientConfig.DisableTCP = false
	clientConfig.DisableUTP = false
	clientConfig.DisableIPv4 = false
	clientConfig.DisableIPv6 = false
	clientConfig.DisableAcceptRateLimiting = true
	clientConfig.NoDHT = false
	clientConfig.Debug = false
	clientConfig.Seed = false
	// clientConfig.PublicIp4 = ""
	// clientConfig.PublicIp6 =  net.IP

	var stop missinggo.SynchronizedEvent
	defer func() {
		stop.Set()
	}()

	client, err := torrent.NewClient(clientConfig)
	if err != nil {
		return xerrors.Errorf("creating client: %v", err)
	}
	defer client.Close()
	go exitSignalHandlers(&stop)
	go func() {
		<-stop.C()
		client.Close()
	}()

	// Write status on the root path on the default HTTP muxer. This will be bound to localhost
	// somewhere if GOPPROF is set, thanks to the envpprof import.
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		client.WriteStatus(w)
	})
	addTorrents(client, torrentname)
	if client.WaitAll() {
		log.Debug("downloaded ALL the torrents")
	} else {
		return xerrors.New("torrent cancelled")
	}
	// if flags.Seed {
	// 	outputStats(client)
	// 	<-stop.C()
	// }
	// outputStats(client)
	return nil
}

// func outputStats(cl *torrent.Client) {
// 	if !statsEnabled() {
// 		return
// 	}
// 	expvar.Do(func(kv expvar.KeyValue) {
// 		fmt.Printf("%s: %s\n", kv.Key, kv.Value)
// 	})
// 	cl.WriteStatus(os.Stdout)
// }
