# xonobo

Radio service for Xonotic SMB modded server *and* generic music players, with simply some ogg vorbis files.

## Setup

- Clone repo
- Run `go build`
- Move folder of ogg vorbis audio to this folder, name it `orgy`
- Run `./xonobo-go`, your service is now at http://127.0.0.1:8293/
- You have a live stream at (bare host) http://127.0.0.1:8296
- Use your IP for public access

## Notes for configuration

1. See possible configurations in `config.example.txt`
2. see `./xonoboctl` for controlling the running process

- For config keys that are supposed to have list of items (`oggdirs`, `lists`), just separate items by space
- You can have oggdirs path in any form, they are virtualized; but try to don’t have two oggdirs with the same basename (do `oggdirs ogg ../ogg2 /ogg3`, don’t `oggdirs ogg ../ogg /ogg`)
- Empty string, `0`, `false` and `no` are accepted as “false” value for boolean items
- Always leave a newline at the end of configuration file
- Live reload doesn’t work if you have changed host/ports. Restart the process instead

## Compare to the [Python version](https://codeberg.org/NaitLee/xonobo)

|     | xonobo-go | xonobo-py |
| --- | --- | --- |
| Portability | Build and run with Go | Just works if you have Python |
| Parsing oggs | Goroutines, no (need) cache | Caches, works lazily |
| Server | Multi-thread and efficient | Single thread, lower efficiency |
| Controlling | `./xonoboctl reload` | Restart the service |
| Live Streaming | For generic music players | A specialized web frontend |

The `vendor` directory contains vendored code that is licensed by other developers.
