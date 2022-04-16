package meta

import (
	"crypto/sha1"
	"encoding/hex"
	"path/filepath"
	"reflect"
	"time"

	"github.com/samber/lo"
	"github.com/xgfone/bt/bencode"
	"github.com/xgfone/bt/metainfo"
)

type TorrentInfoFile struct {
	Length    int64
	Path      string
	Attr      string
	PieceRoot []byte
}

type FilePieceInfo struct {
	Index  int
	Offset int
	Hash   []byte
}

func (m *TorrentInfoFile) FirstPiece(info *TorrentInfo) FilePieceInfo {
	idx := lo.IndexOf(info.Files, m)
	if idx < 0 {
		return FilePieceInfo{}
	}
	sum := int64(0)
	for i := range info.Files {
		if i == idx {
			break
		}
		sum += info.Files[i].Length
	}
	pos := sum / info.PieceLength
	off := sum % info.PieceLength
	return FilePieceInfo{
		Index:  int(pos),
		Offset: int(off),
		Hash:   info.Pieces[pos],
	}
}

type TorrentInfo struct {
	Name        string
	PieceLength int64
	Pieces      [][]byte
	Length      int64
	Files       []*TorrentInfoFile
	MetaVersion int
}

type TorrentMetaInfo struct {
	InfoBytes    []byte
	Announce     string
	AnnounceList []string
	Nodes        []metainfo.HostAddress
	URLList      []string
	Comment      string
	CreationDate time.Time
	CreatedBy    string
	Encoding     string
	info         *TorrentInfo
}

func (m *TorrentMetaInfo) MustInfo() (ti *TorrentInfo) {
	ti, err := m.Info()
	if err != nil {
		panic(err)
	}
	return
}

func (m *TorrentMetaInfo) Info() (ti *TorrentInfo, err error) {
	if m.info != nil {
		return m.info, nil
	}
	var info metainfo.Info
	err = bencode.DecodeBytes(m.InfoBytes, &info)
	if err != nil {
		return
	}

	ti = &TorrentInfo{
		Name:        info.Name,
		PieceLength: info.PieceLength,
		Pieces: lo.Map(info.Pieces, func(t metainfo.Hash, i int) []byte {
			return t.Bytes()
		}),
		Length: info.TotalLength(),
		Files: lo.Map(info.Files, func(t metainfo.File, i int) *TorrentInfoFile {
			return &TorrentInfoFile{
				Length: t.Length,
				Path:   filepath.Join(t.Paths...),
			}
		}),
	}

	mm := map[string]interface{}{}
	err = bencode.DecodeBytes(m.InfoBytes, &mm)
	if err != nil {
		return
	}
	if v, ok := mm["meta version"]; ok {
		ti.MetaVersion = int(reflect.ValueOf(v).Int())
	}

	m.info = ti
	return
}

func (m *TorrentMetaInfo) InfoHash() string {
	hash := sha1.New()
	hash.Write(m.InfoBytes)
	return hex.EncodeToString(hash.Sum(nil))
}
