# prom2timescale
Migrate historical data from Prometheus to TimescaleDB

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
$ curl -XPOST http://localhost:9090/api/v1/admin/tsdb/snapshot
{"status":"success","data":{"name":"20200928T034434Z-23ba198f9608ec46"}}

$ prom2timescale \
    -snapshot-path=/path/to/snapshots/20200928T034434Z-23ba198f9608ec46 \
    -label-key="__name__" \
    -label-value="up" \
    -db-name=PROM_TS_DB_NAME \
    -db-host=PROM_TS_DB_HOST \
    -db-user=PROM_TS_DB_USER \
    -db-password=PROM_TS_DB_PASSWORD
```

## Credits
This works based on [prometheus-tsdb-dump](https://github.com/ryotarai/prometheus-tsdb-dump) and [timescale-prometheus](github.com/timescale/timescale-prometheus)
