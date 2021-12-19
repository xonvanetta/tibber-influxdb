# tibber-influxdb

This will write data points to influxdb based on consumption and currentPrice. The points are written to influxdb with the timestamp from tibber.
Writes to influxdb using v2 API.

Visit https://tibber.com for more information

For API documentation and demo access token, visit https://developer.tibber.com/explorer

## Build
```
go build
```

## Configure and run

### Help
```
Usage of ./tibber-influxdb:
  -influxdb-bucket
    	Change value of InfluxDB-Bucket.
  -influxdb-org
    	Change value of InfluxDB-Org.
  -influxdb-token
    	Change value of InfluxDB-Token.
  -influxdb-url
    	Change value of InfluxDB-Url.
  -interval
    	Change value of Interval. (default 1h0m0s)
  -loglevel
    	Change value of LogLevel. (default info)
  -port
    	Change value of Port. (default 9501)
  -tibber-endpoint
    	Change value of Tibber-Endpoint. (default https://api.tibber.com/v1-beta/gql)
  -tibber-token
    	Change value of Tibber-Token.

Generated environment variables:
   CONFIG_INFLUXDB_BUCKET
   CONFIG_INFLUXDB_ORG
   CONFIG_INFLUXDB_TOKEN
   CONFIG_INFLUXDB_URL
   CONFIG_INTERVAL
   CONFIG_LOGLEVEL
   CONFIG_PORT
   CONFIG_TIBBER_ENDPOINT
   CONFIG_TIBBER_TOKEN

flag: help requested
```