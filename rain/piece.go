package rain

import (
	"crypto/sha1"
	"errors"
	"os"
	"time"
)

type piece struct {
	index      int32 // piece index in whole torrent
	sha1       [sha1.Size]byte
	length     int32          // last piece may not be complete
	targets    []*writeTarget // the place to write downloaded bytes
	blocks     []*block
	bitField   bitField // blocks we have
	haveC      chan *peerConn
	pieceC     chan *peerPieceMessage
	downloaded bool
	log        logger
}

type writeTarget struct {
	file   *os.File
	offset int64
	length int32
}

type block struct {
	index  int32 // block index in piece
	length int32
	data   []byte
}

func (b *block) requestFrom(p *peerConn) error {
	p.log.Debugf("Requsting block #%d from peer %s", b.index, p.conn.RemoteAddr())
	r := newPeerRequestMessage(b.index, b.length)
	return r.send(p.conn)
}

func (p *piece) run() {
	// TODO download blocks
	for {
		select {
		case peer := <-p.haveC:
			if p.downloaded {
				p.log.Debug("Piece is already downloaded")
				break
			}

			unchokeC, err := peer.beInterested()
			if err != nil {
				p.log.Error(err)
				break
			}

			select {
			case <-unchokeC:
				for _, b := range p.blocks {
					if err := b.requestFrom(peer); err != nil {
						p.log.Error(err)
						break
					}
				}
			case <-time.After(time.Minute):
				p.log.Info("Peer did not unchoke")
			}
		case piece := <-p.pieceC:
			p.log.Noticeln("received piece", len(piece.Block))
			// TODO write block to disk
			// piece.
		}
	}

	// TODO hash check

	// TODO write downloaded piece
	// _, err := p.write(nil)
	// if err != nil {
	// 	panic(err)
	// }
}

func (p *piece) Write(b []byte) (n int, err error) {
	if int32(len(b)) != p.length {
		err = errors.New("invalid piece length")
		return
	}
	var m int
	for _, t := range p.targets {
		m, err = t.file.WriteAt(b[n:t.length], t.offset)
		n += m
		if err != nil {
			return
		}
	}
	return
}
