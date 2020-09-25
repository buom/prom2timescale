package main

import (
	"encoding/json"
	"io/ioutil"
	"flag"
	"math"
	"os"
	"strings"

	gokitlog "github.com/go-kit/kit/log"
	"github.com/pkg/errors"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/tsdb"
	"github.com/prometheus/prometheus/tsdb/chunkenc"
	"github.com/prometheus/prometheus/tsdb/chunks"
	"github.com/timescale/timescale-prometheus/pkg/prompb"
	"github.com/timescale/timescale-prometheus/pkg/pgmodel"
	"github.com/timescale/timescale-prometheus/pkg/pgclient"
	"github.com/timescale/timescale-prometheus/pkg/log"
	"github.com/jamiealquiza/envy"
)

type Config struct {
	SnapshotPath		string
	LabelKey			string
	LabelValue 			string
	ExternalLabels		string
	MinTimestamp		int64
	MaxTimestamp		int64
	PgmodelCfg			pgclient.Config
}

func main() {
	logLevel := flag.String("log-level", "debug", "The log level to use [ \"error\", \"warn\", \"info\", \"debug\" ].")
	log.Init(*logLevel)

	cfg := &Config{}
	pgclient.ParseFlags(&cfg.PgmodelCfg)

	flag.StringVar(&cfg.SnapshotPath, "snapshot-path", "", "Path to snapshot directory")
	flag.StringVar(&cfg.LabelKey, "label-key", "", "")
	flag.StringVar(&cfg.LabelValue, "label-value", "", "")
	flag.StringVar(&cfg.ExternalLabels, "external-labels", "{}", "Labels to be added to dumped result in JSON")
	flag.Int64Var(&cfg.MinTimestamp, "min-timestamp", 0, "min of timestamp of datapoints to be dumped; unix time in msec")
	flag.Int64Var(&cfg.MaxTimestamp, "max-timestamp", math.MaxInt64, "min of timestamp of datapoints to be dumped; unix time in msec")
	envy.Parse("PROM_TS")
	flag.Parse()

	if err := run(cfg); err != nil {
		log.Error("error: %s", err)
	}
}

func run(cfg *Config) error {
	snapshotPath := cfg.SnapshotPath
	labelKey := cfg.LabelKey
	labelValue := cfg.LabelValue
	minTimestamp := cfg.MinTimestamp
	maxTimestamp := cfg.MaxTimestamp
	externalLabelsJSON := cfg.ExternalLabels

	externalLabelsMap := map[string]string{}
	if err := json.NewDecoder(strings.NewReader(externalLabelsJSON)).Decode(&externalLabelsMap); err != nil {
		return errors.Wrap(err, "decode external labels")
	}
	var externalLabels labels.Labels
	for k, v := range externalLabelsMap {
		externalLabels = append(externalLabels, labels.Label{Name: k, Value: v})
	}

	client, err := pgclient.NewClient(&cfg.PgmodelCfg, nil)
	if err != nil {
		errors.Wrap(err, "pgclient.NewClient")
	}
	defer client.Close()

	logger := gokitlog.NewLogfmtLogger(os.Stderr)

	fileInfo, _ := ioutil.ReadDir(snapshotPath)
	for _, file := range fileInfo {
		blockPath := snapshotPath + "/" + file.Name()
		log.Info(blockPath)
		block, err := tsdb.OpenBlock(logger, blockPath, chunkenc.NewPool())
		if err != nil {
			return errors.Wrap(err, "tsdb.OpenBlock")
		}

		indexr, err := block.Index()
		if err != nil {
			return errors.Wrap(err, "block.Index")
		}
		defer indexr.Close()

		chunkr, err := block.Chunks()
		if err != nil {
			return errors.Wrap(err, "block.Chunks")
		}
		defer chunkr.Close()

		postings, err := indexr.Postings(labelKey, labelValue)
		if err != nil {
			return errors.Wrap(err, "indexr.Postings")
		}

		var it chunkenc.Iterator
		for postings.Next() {
			ref := postings.At()
			lset := labels.Labels{}
			chks := []chunks.Meta{}
			if err := indexr.Series(ref, &lset, &chks); err != nil {
				return errors.Wrap(err, "indexr.Series")
			}
			if len(externalLabels) > 0 {
				lset = append(lset, externalLabels...)
			}

			for _, meta := range chks {
				chunk, err := chunkr.Chunk(meta.Ref)
				if err != nil {
					return errors.Wrap(err, "chunkr.Chunk")
				}

				var timestamps []int64
				var values []float64

				it := chunk.Iterator(it)
				for it.Next() {
					t, v := it.At()
					if math.IsNaN(v) {
						continue
					}
					if math.IsInf(v, -1) || math.IsInf(v, 1) {
						continue
					}
					if t < minTimestamp || maxTimestamp < t {
						continue
					}
					timestamps = append(timestamps, t)
					values = append(values, v)
				}
				if it.Err() != nil {
					return errors.Wrap(err, "iterator.Err")
				}

				if len(timestamps) == 0 {
					continue
				}

				var labels []prompb.Label
				for _, l := range lset {
					labels = append(labels, prompb.Label{Name: l.Name, Value: l.Value})
				}

				var samples []prompb.Sample
				for i, _ := range timestamps {
					samples = append(samples, prompb.Sample{Timestamp: timestamps[i], Value: values[i]})
				}

				tts := []prompb.TimeSeries{
					{
						Labels: labels,
						Samples: samples,
					},
				}
				req := pgmodel.NewWriteRequest()
				numSamples, err := client.Ingest(tts, req)

				if err != nil {
					log.Warn("msg", "Error sending samples to remote storage", "err", err, "num_samples", numSamples)
				}
			}
		}

		if postings.Err() != nil {
			return errors.Wrap(err, "postings.Err")
		}
	}

	return nil
}
