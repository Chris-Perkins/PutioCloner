# PutioCloner

Clones everything in put.io to a local directory. Multithreaded, chunked downloading support.
Perfect for doing legitimate things with media you obtain legitimately and have on put.io

Put all those home videos you own the rights to on a plex server that hosts these files directly!
Save put.io space by keeping everything locally instead!

Sample usage
```sh
> go run main.go --putio-token <OAuth2 Token>
```

```
$ go run main.go --help
Usage of C:\Cloner\main.exe:
  -chunk-size int
        The chunk size to use when downloading files, default 5 MB
  -max-concurrent int
        The maximum number of concurrent downloads (default 3)
  -out string
        The download location for putio files, default $WorkingDirectory>/Downloads
  -putio-token string
        Required, the OAuth2 token for your Put.io account
  -refresh-rate int
        How often this application should run its loops in seconds (default 30)
  -registry string
        The location of the local file registry (default ".registry")
  -requests string
        The location of pending download requests (default ".requests")
```