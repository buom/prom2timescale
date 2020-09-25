# prom2timescale
migrate historical data from Prometheus to TimescaleDB

## Installation

```
$ git clone https://github.com/buom/prom2timescale.git
$ cd prom2timescale
$ make build
```

## Usage

```
$ prom2timescale -h
```

## Example

```
$ prom2timescale -snapshot-path=/path/to/prometheus-snapshot-dir/ -label-key="__name__" -label-value="up" -db-name=PROM_TS_DB_NAME -db-host=PROM_TS_DB_HOST -db-user=PROM_TS_DB_USER -db-password=PROM_TS_DB_PASSWORD
```

## Credits
This works based on [prometheus-tsdb-dump](https://github.com/ryotarai/prometheus-tsdb-dump) and [timescale-prometheus](github.com/timescale/timescale-prometheus)