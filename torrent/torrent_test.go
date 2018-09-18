package torrent_test

import (
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/cenkalti/log"
	"github.com/crosbymichael/tracker/registry/inmem"
	"github.com/crosbymichael/tracker/server"

	"github.com/cenkalti/rain/internal/logger"
	"github.com/cenkalti/rain/torrent"
)

var (
	trackerAddr    = ":5000"
	torrentFile    = filepath.Join("testdata", "10mb.torrent")
	torrentDataDir = "testdata"
	torrentName    = "10mb"
	timeout        = 10 * time.Second
)

func init() {
	logger.SetLogLevel(log.DEBUG)
}

func xTestDownload(t *testing.T) {
	where, err := ioutil.TempDir("", "rain-")
	if err != nil {
		t.Fatal(err)
	}

	startTracker(t)

	f, err := os.Open(torrentFile)
	if err != nil {
		t.Fatal(err)
	}

	t1, err := torrent.New(f, torrentDataDir, 6881)
	if err != nil {
		t.Fatal(err)
	}
	defer t1.Close()
	t1.Start()

	// Wait for seeder to announce to tracker.
	time.Sleep(100 * time.Millisecond)

	f.Seek(0, io.SeekStart)
	t2, err := torrent.New(f, where, 6882)
	if err != nil {
		t.Fatal(err)
	}
	defer t2.Close()
	t2.Start()

	select {
	case <-t2.NotifyComplete():
	case err = <-t2.NotifyError():
		t.Fatal(err)
	case <-time.After(timeout):
		panic("download did not finish")
	}

	cmd := exec.Command("diff", "-rq",
		filepath.Join(torrentDataDir, torrentName),
		filepath.Join(where, torrentName))
	err = cmd.Run()
	if err != nil {
		t.Fatal(err)
	}

	err = os.RemoveAll(where)
	if err != nil {
		t.Fatal(err)
	}
}

func startTracker(t *testing.T) {
	logger := logrus.New()
	logger.Level = logrus.DebugLevel
	registry := inmem.New()
	s := server.New(120, 30, registry, logger)
	l, err := net.Listen("tcp", trackerAddr)
	if err != nil {
		t.Fatal(err)
	}
	go http.Serve(l, s)
}

func TestDownloadMagnet(t *testing.T) {
	where, err := ioutil.TempDir("", "rain-")
	if err != nil {
		t.Fatal(err)
	}

	startTracker(t)

	f, err := os.Open(torrentFile)
	if err != nil {
		t.Fatal(err)
	}

	t1, err := torrent.New(f, torrentDataDir, 6881)
	if err != nil {
		t.Fatal(err)
	}
	defer t1.Close()
	t1.Start()

	// Wait for seeder to announce to tracker.
	time.Sleep(100 * time.Millisecond)

	f.Seek(0, io.SeekStart)
	magnetLink := "magnet:?xt=urn:btih:0a8e2e8c9371a91e9047ed189ceffbc460803262&dn=10mb&tr=http%3A%2F%2F127.0.0.1%3A5000%2Fannounce"
	t2, err := torrent.NewMagnet(magnetLink, where, 6882)
	if err != nil {
		t.Fatal(err)
	}
	defer t2.Close()
	t2.Start()

	select {
	case <-t2.NotifyComplete():
	case err = <-t2.NotifyError():
		t.Fatal(err)
	case <-time.After(timeout):
		panic("download did not finish")
	}

	cmd := exec.Command("diff", "-rq",
		filepath.Join(torrentDataDir, torrentName),
		filepath.Join(where, torrentName))
	err = cmd.Run()
	if err != nil {
		t.Fatal(err)
	}

	err = os.RemoveAll(where)
	if err != nil {
		t.Fatal(err)
	}
}
