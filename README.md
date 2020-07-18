# putio getter

Go service for fetching and unzipping files from [putio](https://put.io).

# Configuration

The configuration file is read from `${XDG_CONFIG_HOME:-${XDG_CONFIG_DIRS}}/putio/config.json`, and automatically created in `${XDG_CONFIG_HOME}` if one doesn't exist in any searched path.

The default structure is:

```
{
  "OauthToken": "",
  "Downloading": "",
  "Unpacking": "",
  "Interval": "",
  "LogLevel": ""
}
```

`OauthToken` being a token you need to get from [putio](https://app.put.io/settings/account/oauth/apps/new).
`Downloading` and `Unpacking` need to be directories the user the service is running as can read and write.
`Interval` is the frequency with which the service should look for new files to download, e.g. `30s`, `2.5m`, `1h`, etc.
`LogLevel` is the logging level, with values between `debug` and `error` being valid.
