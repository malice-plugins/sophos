# To update the AV run the following:

```bash
$ docker run --name=sophos malice/sophos update
```

## Then to use the updated sophos container:

```bash
$ docker commit sophos malice/sophos:updated
$ docker rm sophos # clean up updated container
$ docker run --rm malice/sophos:updated EICAR
```
