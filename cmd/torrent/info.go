package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"time"

	"github.com/dop251/goja"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/urfave/cli/v2"
	"github.com/wenerme/torrentutils/pkg/meta"
	"github.com/xgfone/bt/bencode"
	"github.com/xgfone/bt/metainfo"
	"golang.org/x/exp/constraints"
	"gopkg.in/yaml.v3"
)

func info(cc *cli.Context) (err error) {
	var all []*meta.TorrentMetaInfo
	byHash := map[string][]byte{}
	for _, v := range cc.Args().Slice() {
		m := map[string]interface{}{}
		data, err := os.ReadFile(v)
		if err != nil {
			return err
		}
		if err = bencode.DecodeBytes(data, &m); err != nil {
			return err
		}
		mi, err := metainfo.Load(bytes.NewReader(data))
		if err != nil {
			return errors.Wrapf(err, "parse torrent: %q", v)
		}
		if v, ok := m["saved by"].(string); ok {
			mi.CreatedBy = v
		}
		if _, ok := m["save date"]; ok {
			mi.CreationDate = reflect.ValueOf(m["save date"]).Int()
		}
		tmr := &meta.TorrentMetaInfo{
			InfoBytes:    mi.InfoBytes,
			Announce:     mi.Announce,
			AnnounceList: lo.Flatten(mi.AnnounceList),
			Nodes:        mi.Nodes,
			URLList:      mi.URLList,
			Comment:      mi.Comment,
			CreationDate: time.Time{},
			CreatedBy:    mi.CreatedBy,
			Encoding:     mi.Encoding,
		}

		sort.Strings(tmr.AnnounceList)
		sort.Strings(tmr.URLList)
		if mi.CreationDate != 0 {
			tmr.CreationDate = time.Unix(mi.CreationDate, 0)
		}
		byHash[tmr.InfoHash()] = data
		all = append(all, tmr)
	}

	f := cc.String("filter")
	if f != "" {
		vm := goja.New()
		if _, err = vm.RunString(`function filter(meta,index){return ` + f + `}`); err != nil {
			return errors.Wrap(err, "run filter")
		}
		var fn func(metaInfo *meta.TorrentMetaInfo, idx int) bool
		if err = vm.ExportTo(vm.Get("filter"), &fn); err != nil {
			return errors.Wrap(err, "export filter")
		}
		all = lo.Filter(all, fn)
	}

	output := cc.String("output")
	var obj interface{}
	if len(all) == 1 {
		v := *all[0]
		v.InfoBytes = nil
		obj = v
	} else if len(all) > 1 {
		obj = all
	}
	var w io.Writer = os.Stdout
	switch {
	case output == "":
		if !cc.Bool("summary-only") {
			for _, v := range all {
				err := tpl.ExecuteTemplate(w, "TorrentMetaInfo", v)
				if err != nil {
					return err
				}
			}
		}

		if cc.Bool("summary") {
			sum := &TorrentMetaInfoSummary{
				Items: all,
				Length: Sum(lo.Map(all, func(v *meta.TorrentMetaInfo, i int) int64 {
					return v.MustInfo().Length
				})),
			}
			err = tpl.ExecuteTemplate(w, "TorrentMetaInfoSummary", sum)
		}
	case output == "info:json":
		if len(all) == 1 {
			m := map[string]interface{}{}
			err = bencode.DecodeBytes(all[0].InfoBytes, &m)
			delete(m, "pieces")

			o, _ := json.MarshalIndent(m, "", "  ")
			_, _ = w.Write(o)
			_, _ = fmt.Fprintln(w)
		}
	case output == "json":
		if len(all) == 1 {
			m := map[string]interface{}{}
			err = bencode.DecodeBytes(all[0].InfoBytes, &m)
			delete(m, "info")
			delete(m, "pieces")
			o, _ := json.MarshalIndent(m, "", "  ")
			_, _ = w.Write(o)
			_, _ = fmt.Fprintln(w)
		}

	case output == "yaml":
		if obj != nil {
			o, err := yaml.Marshal(obj)
			if err != nil {
				return err
			}
			_, _ = w.Write(o)
			_, _ = fmt.Fprintln(w)
		}
	default:
		err = errors.Errorf("output not supported: %q", output)
	}
	return
}

func Sum[V constraints.Integer](a []V) V {
	sum := V(0)
	for _, v := range a {
		sum += v
	}
	return sum
}

type TorrentMetaInfoSummary struct {
	Items  []*meta.TorrentMetaInfo
	Length int64
}

func (v TorrentMetaInfoSummary) CountFiles() int {
	return Sum(lo.Map(v.Items, func(t *meta.TorrentMetaInfo, i int) int {
		return len(t.MustInfo().Files)
	}))
}
