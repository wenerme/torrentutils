package main

import (
	"embed"
	"encoding/hex"
	"math"
	"os"
	"reflect"
	"text/template"

	"github.com/Masterminds/sprig"
	humanize "github.com/dustin/go-humanize"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	cli "github.com/urfave/cli/v2"
)

//go:embed tpl
var tplFS embed.FS
var tpl *template.Template

func main() {
	log.Logger = log.Output(zerolog.NewConsoleWriter()).Output(os.Stderr).With().Stack().Caller().Logger()

	app := &cli.App{
		Name: "torrent",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "log-level",
				Value: "warn",
			},
		},
		Before: func(cc *cli.Context) (err error) {
			if level, err := zerolog.ParseLevel(cc.String("log-level")); err == nil {
				log.Logger = log.Level(level)
			}
			tpl = template.New("torrent")
			tpl.Funcs(sprig.TxtFuncMap())
			tpl.Funcs(map[string]any{
				"bytes": func(v interface{}) string {
					return humanize.Bytes(uint64(reflect.ValueOf(v).Int()))
				},
				"ibytes": func(v interface{}) string {
					return humanize.IBytes(uint64(reflect.ValueOf(v).Int()))
				},
				"log2": func(v interface{}) float64 {
					return math.Log2(reflect.ValueOf(v).Float())
				},
				"hex": func(v interface{}) string {
					return hex.EncodeToString(v.([]byte))
				},
			})
			tpl, err = tpl.ParseFS(tplFS, "**/*.gotpl")
			return
		},
		Commands: cli.Commands{
			{
				Name:    "info",
				Aliases: []string{"i"},
				Usage:   "Show torrent info",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
					},
					&cli.StringFlag{
						Name:    "filter",
						Aliases: []string{"f"},
					},
					&cli.BoolFlag{
						Name:    "summary",
						Aliases: []string{"s"},
					},
					&cli.BoolFlag{
						Name:    "summary-only",
						Aliases: []string{"so"},
					},
				},
				Action: info,
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
}
